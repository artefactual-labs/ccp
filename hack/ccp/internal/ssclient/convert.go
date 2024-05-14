package ssclient

import (
	"github.com/google/uuid"
	"go.artefactual.dev/ssclient/kiota/models"
	"go.artefactual.dev/tools/ref"

	"github.com/artefactual/archivematica/hack/ccp/internal/derrors"
	"github.com/artefactual/archivematica/hack/ccp/internal/ssclient/enums"
)

// TODO: why is kiota using ptrs for mandatory fields?

func convertPipeline(m models.Pipelineable) (_ *Pipeline, err error) {
	derrors.Add(&err, "convertPipeline")

	r := &Pipeline{}

	if uid, err := uuid.Parse(ref.DerefZero(m.GetUuid())); err != nil {
		return nil, err
	} else {
		r.ID = uid
	}

	r.URI = ref.DerefZero(m.GetResourceUri())

	return r, nil
}

func convertLocation(m models.Locationable) (_ *Location, err error) {
	derrors.Add(&err, "convertLocation")

	r := &Location{}

	if uid, err := uuid.Parse(ref.DerefZero(m.GetUuid())); err != nil {
		return nil, err
	} else {
		r.ID = uid
	}

	r.URI = ref.DerefZero(m.GetResourceUri())
	r.Path = ref.DerefZero(m.GetPath())
	r.RelativePath = ref.DerefZero(m.GetRelativePath())
	r.Pipelines = m.GetPipeline()
	r.Purpose = enums.LocationPurpose(ref.DerefDefault(m.GetPurpose(), -1))

	return r, nil
}
