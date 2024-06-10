package controller

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"

	"github.com/google/uuid"

	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
	"github.com/artefactual/archivematica/hack/ccp/internal/derrors"
	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

// jobDecider is a type of job that handles decisions.
//
// It is implemented by outputDecisionJob, updateContextDecisionJob, and
// nextChainDecisionJob. The decide method is executed by the controller to
// propagate the resolution so a job can react to it, e.g. update the context.
// Implementors must mark the job as complete using the markComplete method.
type jobDecider interface {
	decide(ctx context.Context, c choice) error
}

// errWait is used by decision jobs to signal the awaiting condition. The
// controller is expected to consume and resolve it.
type errWait struct {
	decision *decision
}

func (err errWait) Error() string {
	return "errWait"
}

func isErrWait(err error) (*errWait, bool) {
	ew := &errWait{}
	if ok := errors.As(err, &ew); ok {
		return ew, true
	}
	return nil, false
}

// createAwait returns an errWait with all the details needed for the controller to
// coordinate the decision and its resolution.
func createAwait(j *job, choices []choice) (_ uuid.UUID, err error) {
	jd, ok := j.jobRunner.(jobDecider)
	if !ok {
		return uuid.Nil, errors.New("impossible to await this job because it's not a decider")
	}

	err = &errWait{
		decision: newDecision(j.wl.Description.String(), j.pkg, choices, j.id, jd),
	}

	return uuid.Nil, err
}

// choice is a single selectable user decision created by a job to be presented
// within a decision.
type choice struct {
	// label is the string representation of this choice (mandatory).
	label string // TODO: use the i18n field in the workflow package.

	// value is optional, not used by nextChainDecisionJob.
	// - outputClientScriptJob populates a single value, e.g.: `[2]string{"", item.URI}`.
	// - udpateContextDecisionJob populates a pair, e.g.: `[2]string{"AIPCompressionLevel", "1"}`.
	value [2]string

	// nextLink indicates where to continue processing when the decision is
	// resolved using this choice (mandatory).
	nextLink uuid.UUID
}

func (c choice) String() string {
	return fmt.Sprintf("choice %s: %s", c.label, c.value)
}

// A decision can be awaited until someone else resolves it. It provides the
// list of available choices and the resolution interface.
type decision struct {
	id           uuid.UUID  // Identifier of the decision.
	name         string     // Name of the decision.
	pkg          *Package   // Related package.
	choices      []choice   // Ordered list of choices.
	jobID        uuid.UUID  // Identifier of the job.
	decider      jobDecider // So we can call the decide callback.
	res          chan int   // Resolution channel - receives the position of the choice.
	resolved     bool       // Remembers if this decision is already resolved.
	sync.RWMutex            // Protects the decision from concurrent read-writes.
}

func newDecision(name string, pkg *Package, choices []choice, jobID uuid.UUID, job jobDecider) *decision {
	return &decision{
		id:      uuid.New(),
		name:    name,
		pkg:     pkg,
		choices: choices,
		jobID:   jobID,
		decider: job,

		// The channel is buffered so the decision can be resolved even when
		// there is no one awaiting.
		res: make(chan int, 1),
	}
}

// resolve the decision given the position of one of the known choices.
func (d *decision) resolve(pos int) error {
	d.RLock()
	if d.resolved {
		return errors.New("decision is not pending resolution")
	}
	d.RUnlock()

	d.Lock()
	d.res <- pos
	d.resolved = true
	d.Unlock()

	return nil
}

// await waits for the resolution. It returns the next workflow chain link,
// which will most likely be uuid.Nil unless indicated by nextChainDecisionJob.
func (d *decision) await(ctx context.Context) (uuid.UUID, error) {
	select {
	case <-ctx.Done():
		return uuid.Nil, ctx.Err()
	case pos := <-d.res:
		ln := len(d.choices)
		if ln < 0 || ln > len(d.choices) {
			return uuid.Nil, errors.New("unavailable choice")
		}
		choice := d.choices[pos]
		if err := d.decider.decide(ctx, choice); err != nil {
			return uuid.Nil, err
		}
		return choice.nextLink, nil
	}
}

func (d *decision) convert() *adminv1.Decision {
	d.RLock()
	defer d.RUnlock()

	ret := &adminv1.Decision{
		Id:          d.id.String(),
		Name:        d.name,
		Choice:      make([]*adminv1.Choice, 0, len(d.choices)),
		PackageId:   d.pkg.id.String(),
		PackagePath: d.pkg.PathForDB(),
		PackageType: d.pkg.packageType().String(),
		JobId:       d.jobID.String(),
	}

	for i, item := range d.choices {
		ret.Choice = append(ret.Choice, &adminv1.Choice{
			Id:    int32(i),
			Label: item.label,
		})
	}

	return ret
}

// outputDecisionJob.
//
// A job that handles a workflow decision point, with choices based on script
// output.
//
// Manager: linkTaskManagerGetUserChoiceFromMicroserviceGeneratedList.
// Class: OutputDecisionJob(DecisionJob).
type outputDecisionJob struct {
	j      *job
	config *workflow.LinkStandardTaskConfig
}

var (
	_ jobRunner  = (*outputDecisionJob)(nil)
	_ jobDecider = (*outputDecisionJob)(nil)
)

func newOutputDecisionJob(j *job) (*outputDecisionJob, error) {
	ret := &outputDecisionJob{
		j:      j,
		config: &workflow.LinkStandardTaskConfig{},
	}
	if err := loadConfig(j.wl, ret.config); err != nil {
		return nil, err
	}

	return ret, nil
}

func (l *outputDecisionJob) exec(ctx context.Context) (_ uuid.UUID, err error) {
	derrors.Add(&err, "outputDecisionJob")

	nextLink := exitCodeLinkID(l.j.wl, 0)

	var c *choice
	locURI, err := l.j.pkg.PreconfiguredChoice(l.j.wl.ID)
	if err != nil {
		return uuid.Nil, err
	} else if locURI != "" {
		for _, item := range l.j.chain.choices {
			if locURI == item.value[1] {
				c = &item
				break
			}
		}
	}
	if c != nil {
		return nextLink, l.decide(ctx, *c)
	}

	// Store the next link in all choices we're sharing with the decision.
	for i := range l.j.chain.choices {
		l.j.chain.choices[i].nextLink = nextLink
	}

	return createAwait(l.j, l.j.chain.choices)
}

func (l *outputDecisionJob) decide(ctx context.Context, c choice) error {
	// Pass the choice to the next job. This case is only used to select an AIP
	// store URI, and the value of execute (script_name here) is a replacement
	// string (e.g. %AIPsStore%).
	l.j.chain.context.Set(l.config.Execute, c.value[1])

	return l.j.markComplete(ctx)
}

// nextChainDecisionJob.
//
// A type of workflow decision that determines the next chain to be executed,
// by UUID.
//
// Manager: linkTaskManagerChoice.
// Class: NextChainDecisionJob(DecisionJob).
type nextChainDecisionJob struct {
	j      *job
	config *workflow.LinkMicroServiceChainChoice
}

var (
	_ jobRunner  = (*nextChainDecisionJob)(nil)
	_ jobDecider = (*outputDecisionJob)(nil)
)

func newNextChainDecisionJob(j *job) (*nextChainDecisionJob, error) {
	ret := &nextChainDecisionJob{
		j:      j,
		config: &workflow.LinkMicroServiceChainChoice{},
	}
	if err := loadConfig(j.wl, ret.config); err != nil {
		return nil, err
	}

	return ret, nil
}

func (l *nextChainDecisionJob) exec(ctx context.Context) (_ uuid.UUID, err error) {
	derrors.Add(&err, "nextChainDecisionJob")

	// Use a preconfigured choice if it validates.
	chainID, err := l.j.pkg.PreconfiguredChoice(l.j.wl.ID)
	if err != nil {
		return uuid.Nil, err
	} else if chainID != "" {
		cid, err := uuid.Parse(chainID)
		if err != nil {
			return uuid.Nil, err
		}

		// Fail if the choice is not available in workflow.
		var matched bool
		for _, item := range l.config.Choices {
			if _, ok := l.j.wf.Chains[item]; ok {
				matched = true
			}
		}
		if !matched {
			return uuid.Nil, fmt.Errorf("choice %s is not one of the available choices", chainID)
		}
		return cid, nil
	}

	// Build choices.
	choices := make([]choice, 0, len(l.config.Choices))
	for _, item := range l.config.Choices {
		c := choice{}
		ch, ok := l.j.wf.Chains[item]
		if !ok {
			continue
		}
		if !workflow.ChoiceAvailable(l.j.wl, ch) {
			continue
		}
		c.label = ch.Description.String()
		c.nextLink = item
		choices = append(choices, c)
	}

	return createAwait(l.j, choices)
}

func (l *nextChainDecisionJob) decide(ctx context.Context, c choice) error { //nolint: unparam
	return l.j.markComplete(ctx)
}

// updateContextDecisionJob.
//
// A job that updates the job chain context based on a user choice.
//
// TODO: This type of job is mostly copied from the previous
// linkTaskManagerReplacementDicFromChoice, and it seems to have multiple ways
// of executing. It could use some cleanup.
//
// Manager: linkTaskManagerReplacementDicFromChoice.
// Class: UpdateContextDecisionJob(DecisionJob) (decisions.py).
type updateContextDecisionJob struct {
	j      *job
	config *workflow.LinkMicroServiceChoiceReplacementDic
}

var (
	_ jobRunner  = (*updateContextDecisionJob)(nil)
	_ jobDecider = (*outputDecisionJob)(nil)
)

// Maps decision point UUIDs and decision UUIDs to their "canonical"
// equivalents. This is useful for when there are multiple decision points which
// are effectively identical and a preconfigured decision for one should hold
// for all of the others as well. For example, there are 5 "Assign UUIDs to
// directories?" decision points and making a processing config decision for the
// designated canonical one, in this case
// 'bd899573-694e-4d33-8c9b-df0af802437d', should result in that decision taking
// effect for all of the others as well. This allows that.
// TODO: this should be defined in the workflow, not hardcoded here.
var updateContextDecisionJobChoiceMapping = map[uuid.UUID]uuid.UUID{
	// Decision point "Assign UUIDs to directories?".
	uuid.MustParse("8882bad4-561c-4126-89c9-f7f0c083d5d7"): uuid.MustParse("bd899573-694e-4d33-8c9b-df0af802437d"),
	uuid.MustParse("e10a31c3-56df-4986-af7e-2794ddfe8686"): uuid.MustParse("bd899573-694e-4d33-8c9b-df0af802437d"),
	uuid.MustParse("d6f6f5db-4cc2-4652-9283-9ec6a6d181e5"): uuid.MustParse("bd899573-694e-4d33-8c9b-df0af802437d"),
	uuid.MustParse("1563f22f-f5f7-4dfe-a926-6ab50d408832"): uuid.MustParse("bd899573-694e-4d33-8c9b-df0af802437d"),
	// Decision "Yes" (for "Assign UUIDs to directories?").
	uuid.MustParse("7e4cf404-e62d-4dc2-8d81-6141e390f66f"): uuid.MustParse("2dc3f487-e4b0-4e07-a4b3-6216ed24ca14"),
	uuid.MustParse("2732a043-b197-4cbc-81ab-4e2bee9b74d3"): uuid.MustParse("2dc3f487-e4b0-4e07-a4b3-6216ed24ca14"),
	uuid.MustParse("aa793efa-1b62-498c-8f92-cab187a99a2a"): uuid.MustParse("2dc3f487-e4b0-4e07-a4b3-6216ed24ca14"),
	uuid.MustParse("efd98ddb-80a6-4206-80bf-81bf00f84416"): uuid.MustParse("2dc3f487-e4b0-4e07-a4b3-6216ed24ca14"),
	// Decision "No" (for "Assign UUIDs to directories?").
	uuid.MustParse("0053c670-3e61-4a3e-a188-3a2dd1eda426"): uuid.MustParse("891f60d0-1ba8-48d3-b39e-dd0934635d29"),
	uuid.MustParse("8e93e523-86bb-47e1-a03a-4b33e13f8c5e"): uuid.MustParse("891f60d0-1ba8-48d3-b39e-dd0934635d29"),
	uuid.MustParse("6dfbeff8-c6b1-435b-833a-ed764229d413"): uuid.MustParse("891f60d0-1ba8-48d3-b39e-dd0934635d29"),
	uuid.MustParse("dc0ee6b6-ed5f-42a3-bc8f-c9c7ead03ed1"): uuid.MustParse("891f60d0-1ba8-48d3-b39e-dd0934635d29"),
}

func newUpdateContextDecisionJob(j *job) (*updateContextDecisionJob, error) {
	ret := &updateContextDecisionJob{
		j:      j,
		config: &workflow.LinkMicroServiceChoiceReplacementDic{},
	}
	if err := loadConfig(j.wl, ret.config); err != nil {
		return nil, err
	}

	return ret, nil
}

func (l *updateContextDecisionJob) exec(ctx context.Context) (linkID uuid.UUID, err error) {
	derrors.Add(&err, "nextChainDecisionJob")
	defer func() {
		if err == nil {
			linkID = exitCodeLinkID(l.j.wl, 0)
		}
	}()

	// Load new context from the database (DashboardSettings).
	// We have two chain links in workflow with no replacements configured:
	// "7f975ba6" and "a0db8294". This feels like a different case where we are
	// loading the replacements from the application database.
	// TODO: split this out?
	if len(l.config.Replacements) == 0 {
		if dict, err := l.loadDatabaseContext(ctx); err != nil {
			return uuid.Nil, fmt.Errorf("load dict from db: %v", err)
		} else if len(dict) > 0 {
			l.j.chain.update(dict)
			return uuid.Nil, nil
		}
	}

	// Load new context from processing configuration.
	if dict, err := l.loadPreconfiguredContext(); err != nil {
		return uuid.Nil, fmt.Errorf("load context with preconfigured choice: %v", err)
	} else if len(dict) > 0 {
		l.j.chain.update(dict)
		return uuid.Nil, nil
	}

	// Build choices.
	choices := make([]choice, len(l.config.Replacements))
	for i, item := range l.config.Replacements {
		c := &choices[i]
		c.label = item.Description.String()
		c.nextLink = exitCodeLinkID(l.j.wl, 0)
		for k, v := range item.Items {
			c.value = [2]string{k, v}
			break
		}
	}

	return createAwait(l.j, choices)
}

// loadDatabaseContext loads the context dictionary from the database.
func (l *updateContextDecisionJob) loadDatabaseContext(ctx context.Context) (map[string]string, error) {
	// We're looking for the "execute" parameter of the next link, e.g.:
	// "upload-archivesspace_v0.0" or "upload-qubit_v0.0".
	ln, ok := l.j.wf.Links[l.j.wl.FallbackLinkID]
	if !ok {
		return nil, nil
	}
	cfg, ok := ln.Config.(workflow.LinkStandardTaskConfig)
	if !ok {
		return nil, nil
	}
	if cfg.Execute == "" {
		return nil, nil
	}

	ret, err := l.j.pkg.store.ReadDict(ctx, cfg.Execute)
	if err != nil {
		return nil, err
	}

	return l.formatChoices(ret), nil
}

// loadPreconfiguredContext loads the context dictionary from the workflow.
func (l *updateContextDecisionJob) loadPreconfiguredContext() (map[string]string, error) {
	var normalizedChoice uuid.UUID
	if v, ok := updateContextDecisionJobChoiceMapping[l.j.wl.ID]; ok {
		normalizedChoice = v
	} else {
		normalizedChoice = l.j.wl.ID
	}

	choices, err := l.j.pkg.parseProcessingConfig()
	if err != nil {
		return nil, err
	}

	ret := map[string]string{}
	for _, choice := range choices {
		if choice.AppliesTo != normalizedChoice.String() {
			continue
		}
		desiredChoice, err := uuid.Parse(choice.GoToChain)
		if err != nil {
			return nil, err
		}
		if v, ok := updateContextDecisionJobChoiceMapping[desiredChoice]; ok {
			desiredChoice = v
		}
		ln, ok := l.j.wf.Links[normalizedChoice]
		if !ok {
			return nil, fmt.Errorf("desired choice not found: %s", desiredChoice)
		}
		config, ok := ln.Config.(workflow.LinkMicroServiceChoiceReplacementDic)
		if !ok {
			return nil, fmt.Errorf("desired choice doesn't have the expected type: %s", desiredChoice)
		}
		for _, replacement := range config.Replacements {
			if replacement.ID == desiredChoice {
				choices := maps.Clone(replacement.Items)
				ret = l.formatChoices(choices)
				break
			}
		}
	}

	return ret, nil
}

func (l *updateContextDecisionJob) formatChoices(choices map[string]string) map[string]string {
	for k, v := range choices {
		delete(choices, k)
		choices[fmt.Sprintf("%%%s%%", k)] = v
	}

	return choices
}

func (l *updateContextDecisionJob) decide(ctx context.Context, c choice) error {
	if c.value[0] != "" {
		l.j.chain.context.Set(fmt.Sprintf("%%%s%%", c.value[0]), c.value[1])
	}

	return l.j.markComplete(ctx)
}
