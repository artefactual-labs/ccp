package controller

import (
	"github.com/google/uuid"

	adminv1 "github.com/artefactual/archivematica/hack/ccp/internal/api/gen/archivematica/ccp/admin/v1beta1"
)

// TransferType represents a known type of transfer in Archivematica.
type TransferType struct {
	// Name of the transfer type, e.g. "standard", "zipfile", etc...
	Name string

	// Type enum value as described by the adminv1 proto.
	Type adminv1.TransferType

	// WatcheDir is the watched directory used to trigger this type of transfer.
	WatchedDir string

	// BypassChainID is the chain bypass used to start a new transfer of this
	// type with auto-approval.
	BypassChainID uuid.UUID

	// BypassLinkID is the specific chain link within the bypass chain where
	// we want to start processing when using auto-approval.
	BypassLinkID uuid.UUID

	// DecisionLink is the chain link used to prompt the user for approval.
	// Decision is the chain that we have to choose to accept the transfer.
	// TODO: remove these two once we implement the Decision API.
	DecisionLink uuid.UUID
	Decision     uuid.UUID
}

type TransferTypes []TransferType

// Decide resolves the workflow decision point that implements the approval.
func (t TransferTypes) Decide(linkID uuid.UUID) uuid.UUID {
	for _, item := range t {
		if item.DecisionLink == linkID {
			return item.Decision
		}
	}
	return uuid.Nil
}

func (t TransferTypes) WithName(name string) *TransferType {
	for _, item := range t {
		if item.Name == name {
			return &item
		}
	}

	return nil
}

func (t TransferTypes) WithType(tt adminv1.TransferType) *TransferType {
	if tt == adminv1.TransferType_TRANSFER_TYPE_UNSPECIFIED {
		tt = adminv1.TransferType_TRANSFER_TYPE_STANDARD
	}

	for _, item := range t {
		if item.Type == tt {
			return &item
		}
	}

	return nil
}

// List of transfer types supported by Archivematica.
var Transfers TransferTypes = []TransferType{
	{
		Name:          "standard",
		Type:          adminv1.TransferType_TRANSFER_TYPE_STANDARD,
		WatchedDir:    "activeTransfers/standardTransfer",
		BypassChainID: uuid.MustParse("6953950b-c101-4f4c-a0c3-0cd0684afe5e"),
		BypassLinkID:  uuid.MustParse("045c43ae-d6cf-44f7-97d6-c8a602748565"),
		DecisionLink:  uuid.MustParse("0c94e6b5-4714-4bec-82c8-e187e0c04d77"),
		Decision:      uuid.MustParse("b4567e89-9fea-4256-99f5-a88987026488"),
	},
	{
		Name:          "zipfile",
		Type:          adminv1.TransferType_TRANSFER_TYPE_ZIP_FILE,
		WatchedDir:    "activeTransfers/zippedDirectory",
		BypassChainID: uuid.MustParse("f3caceff-5ad5-4bad-b98c-e73f8cd03450"),
		BypassLinkID:  uuid.MustParse("541f5994-73b0-45bb-9cb5-367c06a21be7"),
	},
	{
		Name:          "unzipped bag",
		Type:          adminv1.TransferType_TRANSFER_TYPE_UNZIPPED_BAG,
		WatchedDir:    "activeTransfers/baggitDirectory",
		BypassChainID: uuid.MustParse("c75ef451-2040-4511-95ac-3baa0f019b48"),
		BypassLinkID:  uuid.MustParse("154dd501-a344-45a9-97e3-b30093da35f5"),
	},
	{
		Name:          "zipped bag",
		Type:          adminv1.TransferType_TRANSFER_TYPE_ZIPPED_BAG,
		WatchedDir:    "activeTransfers/baggitZippedDirectory",
		BypassChainID: uuid.MustParse("167dc382-4ab1-4051-8e22-e7f1c1bf3e6f"),
		BypassLinkID:  uuid.MustParse("3229e01f-adf3-4294-85f7-4acb01b3fbcf"),
	},
	{
		Name:          "dspace",
		Type:          adminv1.TransferType_TRANSFER_TYPE_DSPACE,
		WatchedDir:    "activeTransfers/Dspace",
		BypassChainID: uuid.MustParse("1cb2ef0e-afe8-45b5-8d8f-a1e120f06605"),
		BypassLinkID:  uuid.MustParse("bda96b35-48c7-44fc-9c9e-d7c5a05016c1"),
	},
	{
		Name:          "maildir",
		Type:          adminv1.TransferType_TRANSFER_TYPE_MAILDIR,
		WatchedDir:    "activeTransfers/maildir",
		BypassChainID: uuid.MustParse("d381cf76-9313-415f-98a1-55c91e4d78e0"),
		BypassLinkID:  uuid.MustParse("da2d650e-8ce3-4b9a-ac97-8ca4744b019f"),
	},
	{
		Name:          "TRIM",
		Type:          adminv1.TransferType_TRANSFER_TYPE_TRIM,
		WatchedDir:    "activeTransfers/TRIM",
		BypassChainID: uuid.MustParse("e4a59e3e-3dba-4eb5-9cf1-c1fb3ae61fa9"),
		BypassLinkID:  uuid.MustParse("2483c25a-ade8-4566-a259-c6c37350d0d6"),
	},
	{
		Name:          "dataverse",
		Type:          adminv1.TransferType_TRANSFER_TYPE_DATAVERSE,
		WatchedDir:    "activeTransfers/dataverseTransfer",
		BypassChainID: uuid.MustParse("10c00bc8-8fc2-419f-b593-cf5518695186"),
		BypassLinkID:  uuid.MustParse("0af6b163-5455-4a76-978b-e35cc9ee445f"),
	},
}
