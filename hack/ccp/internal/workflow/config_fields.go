package workflow

import (
	"context"
	"slices"

	"github.com/google/uuid"

	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
)

type configField struct {
	// Mandatory fields expected in the processingConfigFields list.
	linkID  uuid.UUID
	name    string
	builder fieldBuilder

	wf     *Document
	link   *Link
	cached *adminv1.ProcessingConfigField
}

// build populates shared attributes and runs the builder.
func (c *configField) build(wf *Document) {
	c.wf = wf
	c.link = c.wf.Links[c.linkID]
	c.cached = &adminv1.ProcessingConfigField{
		Id:    c.linkID.String(),
		Name:  c.name,
		Label: i18n(c.link.Description),
	}

	c.builder.build(c)
}

// fieldBuilder is the interface that all processing configuration fields must
// implement to produce the config field to be cached.
type fieldBuilder interface {
	// build is allowed to panic.
	build(cf *configField)
}

// sharedConfigChoicesField ...
type sharedChainChoicesField struct {
	relatedLinks []uuid.UUID
}

var _ fieldBuilder = (*sharedChainChoicesField)(nil)

func (f *sharedChainChoicesField) build(cf *configField) {
	config := cf.link.Config.(LinkMicroServiceChainChoice)

	// Full list of choices based on the master link.
	choices := make([]I18nField, 0, len(config.Choices))
	for _, chainID := range config.Choices {
		chain := cf.wf.Chains[chainID]
		choices = append(choices, chain.Description)
	}

	linkIDs := slices.Concat([]uuid.UUID{cf.linkID}, f.relatedLinks)

	for _, choiceDesc := range choices {
		choice := &adminv1.ProcessingConfigFieldChoice{
			Label: i18n(choiceDesc),
		}
		for _, linkID := range linkIDs {
			link := cf.wf.Links[linkID]
			for _, chainID := range link.Config.(LinkMicroServiceChainChoice).Choices {
				chain := cf.wf.Chains[chainID]
				if chain.Description.String() == choiceDesc.String() {
					choice.AppliesTo = append(choice.AppliesTo, &adminv1.ProcessingConfigFieldChoiceAppliesTo{
						LinkId: linkID.String(),
						Value:  chainID.String(),
						Label:  i18n(chain.Description),
					})
					if linkID == cf.linkID {
						choice.Value = chainID.String()
					}
				}
			}
		}
		cf.cached.Choice = append(cf.cached.Choice, choice)
	}
}

// replaceDictField ...
type replaceDictField struct{}

var _ fieldBuilder = (*replaceDictField)(nil)

func (f *replaceDictField) build(cf *configField) {
	config := cf.link.Config.(LinkMicroServiceChoiceReplacementDic)

	cf.cached.Choice = make([]*adminv1.ProcessingConfigFieldChoice, 0, len(config.Replacements))
	for _, item := range config.Replacements {
		cf.cached.Choice = append(cf.cached.Choice, &adminv1.ProcessingConfigFieldChoice{
			Value: item.ID.String(),
			Label: i18n(item.Description),
			AppliesTo: []*adminv1.ProcessingConfigFieldChoiceAppliesTo{
				{
					LinkId: cf.linkID.String(),
					Value:  item.ID.String(),
					Label:  i18n(cf.link.Description),
				},
			},
		})
	}
}

// chainChoicesField populates choices based on the list of chains indicated by
// `chain_choices` in the workflow link definition.
type chainChoicesField struct {
	// ignoredChoices is an optional list of chain names to be ignored.
	ignoredChoices []string
	// findDuplicates is an optional string used to match all links making use
	// of that choice, e.g. "Normalize for preservation".
	findDuplicates string
}

var _ fieldBuilder = (*chainChoicesField)(nil)

func (f *chainChoicesField) build(cf *configField) {
	config := cf.link.Config.(LinkMicroServiceChainChoice)

	for _, chainID := range config.Choices {
		chain := cf.wf.Chains[chainID]
		chainDesc := chain.Description.String()
		if slices.Contains(f.ignoredChoices, chainDesc) {
			continue
		}
		choice := &adminv1.ProcessingConfigFieldChoice{
			Value: chainID.String(),
			Label: i18n(chain.Description),
			AppliesTo: []*adminv1.ProcessingConfigFieldChoiceAppliesTo{
				{
					LinkId: cf.link.ID.String(),
					Value:  chain.ID.String(),
					Label:  i18n(chain.Description),
				},
			},
		}
		// Incorporate additional AppliesTo objects from duplicates.
		if len(f.findDuplicates) > 0 {
			for id, dup := range cf.wf.Links {
				if id == cf.linkID {
					continue
				}
				if dup.Description.String() != f.findDuplicates {
					continue
				}
				c, ok := dup.Config.(LinkMicroServiceChainChoice)
				if !ok {
					continue
				}
				// Confirmed dup.
				for _, cid := range c.Choices {
					if ch, ok := cf.wf.Chains[cid]; ok {
						if ch.Description.String() == chainDesc {
							choice.AppliesTo = append(choice.AppliesTo, &adminv1.ProcessingConfigFieldChoiceAppliesTo{
								LinkId: dup.ID.String(),
								Value:  ch.ID.String(),
								Label:  i18n(ch.Description),
							})
						}
					}
				}

			}
		}
		cf.cached.Choice = append(cf.cached.Choice, choice)
	}
}

var processingConfigFields []*configField = []*configField{
	{
		name:   "virus_scanning",
		linkID: uuid.MustParse("856d2d65-cd25-49fa-8da9-cabb78292894"),
		builder: &sharedChainChoicesField{
			relatedLinks: []uuid.UUID{
				uuid.MustParse("1dad74a2-95df-4825-bbba-dca8b91d2371"),
				uuid.MustParse("7e81f94e-6441-4430-a12d-76df09181b66"),
				uuid.MustParse("390d6507-5029-4dae-bcd4-ce7178c9b560"),
				uuid.MustParse("97a5ddc0-d4e0-43ac-a571-9722405a0a9b"),
			},
		},
	},
	{
		name:    "assign_uuids_to_directories",
		linkID:  uuid.MustParse("bd899573-694e-4d33-8c9b-df0af802437d"),
		builder: &replaceDictField{},
	},
	{
		name:    "generate_transfer_structure",
		linkID:  uuid.MustParse("56eebd45-5600-4768-a8c2-ec0114555a3d"),
		builder: &chainChoicesField{},
	},
	{
		name:    "select_format_id_tool_transfer",
		linkID:  uuid.MustParse("f09847c2-ee51-429a-9478-a860477f6b8d"),
		builder: &replaceDictField{},
	},
	{
		name:    "extract_packages",
		linkID:  uuid.MustParse("dec97e3c-5598-4b99-b26e-f87a435a6b7f"),
		builder: &chainChoicesField{},
	},
	{
		name:    "delete_packages",
		linkID:  uuid.MustParse("f19926dd-8fb5-4c79-8ade-c83f61f55b40"),
		builder: &replaceDictField{},
	},
	{
		name:    "policy_checks_originals",
		linkID:  uuid.MustParse("70fc7040-d4fb-4d19-a0e6-792387ca1006"),
		builder: &chainChoicesField{},
	},
	{
		name:    "examine_contents",
		linkID:  uuid.MustParse("accea2bf-ba74-4a3a-bb97-614775c74459"),
		builder: &chainChoicesField{},
	},
	{
		name:   "create_sip",
		linkID: uuid.MustParse("bb194013-597c-4e4a-8493-b36d190f8717"),
		builder: &chainChoicesField{
			ignoredChoices: []string{"Reject transfer"},
		},
	},
	{
		name:    "select_format_id_tool_ingest",
		linkID:  uuid.MustParse("7a024896-c4f7-4808-a240-44c87c762bc5"),
		builder: &replaceDictField{},
	},
	{
		name:   "normalize",
		linkID: uuid.MustParse("cb8e5706-e73f-472f-ad9b-d1236af8095f"),
		builder: &chainChoicesField{
			ignoredChoices: []string{"Reject SIP"},
			findDuplicates: "Normalize",
		},
	},
	{
		name:   "normalize_transfer",
		linkID: uuid.MustParse("de909a42-c5b5-46e1-9985-c031b50e9d30"),
		builder: &chainChoicesField{
			ignoredChoices: []string{"Redo", "Reject"},
		},
	},
	{
		name:    "normalize_thumbnail_mode",
		linkID:  uuid.MustParse("498f7a6d-1b8c-431a-aa5d-83f14f3c5e65"),
		builder: &replaceDictField{},
	},
	{
		name:    "policy_checks_preservation_derivatives",
		linkID:  uuid.MustParse("153c5f41-3cfb-47ba-9150-2dd44ebc27df"),
		builder: &chainChoicesField{},
	},
	{
		name:    "policy_checks_access_derivatives",
		linkID:  uuid.MustParse("8ce07e94-6130-4987-96f0-2399ad45c5c2"),
		builder: &chainChoicesField{},
	},
	{
		name:    "bind_pids",
		linkID:  uuid.MustParse("a2ba5278-459a-4638-92d9-38eb1588717d"),
		builder: &chainChoicesField{},
	},
	{
		name:    "normative_structmap",
		linkID:  uuid.MustParse("d0dfa5fc-e3c2-4638-9eda-f96eea1070e0"),
		builder: &chainChoicesField{},
	},
	{
		name:    "reminder",
		linkID:  uuid.MustParse("eeb23509-57e2-4529-8857-9d62525db048"),
		builder: &chainChoicesField{},
	},
	{
		name:    "transcribe_file",
		linkID:  uuid.MustParse("82ee9ad2-2c74-4c7c-853e-e4eaf68fc8b6"),
		builder: &chainChoicesField{},
	},
	{
		name:    "select_format_id_tool_submissiondocs",
		linkID:  uuid.MustParse("087d27be-c719-47d8-9bbb-9a7d8b609c44"),
		builder: &replaceDictField{},
	},
	{
		name:    "compression_algo",
		linkID:  uuid.MustParse("01d64f58-8295-4b7b-9cab-8f1b153a504f"),
		builder: &replaceDictField{},
	},
	{
		name:    "compression_level",
		linkID:  uuid.MustParse("01c651cb-c174-4ba4-b985-1d87a44d6754"),
		builder: &replaceDictField{},
	},
	{
		name:   "store_aip",
		linkID: uuid.MustParse("2d32235c-02d4-4686-88a6-96f4d6c7b1c3"),
		builder: &chainChoicesField{
			ignoredChoices: []string{"Reject AIP"},
		},
	},
	{
		name:    "upload_dip",
		linkID:  uuid.MustParse("92879a29-45bf-4f0b-ac43-e64474f0f2f9"),
		builder: &chainChoicesField{},
	},
	{
		name:    "store_dip",
		linkID:  uuid.MustParse("5e58066d-e113-4383-b20b-f301ed4d751c"),
		builder: &chainChoicesField{},
	},
}

type ProcessingConfigForm struct {
	wf     *Document
	fields []*configField
}

// NewProcessingConfigForms can panic during initialization.
func NewProcessingConfigForm(wf *Document) *ProcessingConfigForm {
	f := &ProcessingConfigForm{
		wf:     wf,
		fields: processingConfigFields, // This is a global.
	}

	for _, item := range f.fields {
		item.build(wf)
	}

	return f
}

func (f *ProcessingConfigForm) Fields(ctx context.Context) ([]*adminv1.ProcessingConfigField, error) {
	fields := make([]*adminv1.ProcessingConfigField, 0, len(f.fields))

	var err error
	for _, cf := range f.fields {
		fields = append(fields, cf.cached)
	}

	return fields, err
}

func i18n(tx I18nField) *adminv1.I18N {
	return &adminv1.I18N{Tx: tx}
}
