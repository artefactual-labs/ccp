package controller

import (
	"github.com/google/uuid"
)

type TransferType struct {
	// WatcheDir is the watched directory used to trigger this type of transfer.
	WatchedDir string

	// Chain is the chain used to start processing if approval is omitted.
	Chain uuid.UUID

	// Link is the link used to start processing if approval is omitted.
	Link uuid.UUID

	// DecisionLink is the chain link used to require user approval.
	DecisionLink uuid.UUID

	// Decision is the approved chain.
	Decision uuid.UUID
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

// List of transfer types supported by Archivematica.
var Transfers TransferTypes = []TransferType{
	{
		WatchedDir:   "activeTransfers/standardTransfer",
		Chain:        uuid.MustParse("6953950b-c101-4f4c-a0c3-0cd0684afe5e"),
		Link:         uuid.MustParse("045c43ae-d6cf-44f7-97d6-c8a602748565"),
		DecisionLink: uuid.MustParse("0c94e6b5-4714-4bec-82c8-e187e0c04d77"),
		Decision:     uuid.MustParse("b4567e89-9fea-4256-99f5-a88987026488"),
	},
	{
		WatchedDir: "activeTransfers/zippedDirectory",
		Chain:      uuid.MustParse("f3caceff-5ad5-4bad-b98c-e73f8cd03450"),
		Link:       uuid.MustParse("541f5994-73b0-45bb-9cb5-367c06a21be7"),
	},
	{
		WatchedDir: "activeTransfers/baggitDirectory",
		Chain:      uuid.MustParse("c75ef451-2040-4511-95ac-3baa0f019b48"),
		Link:       uuid.MustParse("154dd501-a344-45a9-97e3-b30093da35f5"),
	},
	{
		WatchedDir: "activeTransfers/baggitZippedDirectory",
		Chain:      uuid.MustParse("167dc382-4ab1-4051-8e22-e7f1c1bf3e6f"),
		Link:       uuid.MustParse("3229e01f-adf3-4294-85f7-4acb01b3fbcf"),
	},
	{
		WatchedDir: "activeTransfers/Dspace",
		Chain:      uuid.MustParse("1cb2ef0e-afe8-45b5-8d8f-a1e120f06605"),
		Link:       uuid.MustParse("bda96b35-48c7-44fc-9c9e-d7c5a05016c1"),
	},
	{
		WatchedDir: "activeTransfers/maildir",
		Chain:      uuid.MustParse("d381cf76-9313-415f-98a1-55c91e4d78e0"),
		Link:       uuid.MustParse("da2d650e-8ce3-4b9a-ac97-8ca4744b019f"),
	},
	{
		WatchedDir: "activeTransfers/TRIM",
		Chain:      uuid.MustParse("e4a59e3e-3dba-4eb5-9cf1-c1fb3ae61fa9"),
		Link:       uuid.MustParse("2483c25a-ade8-4566-a259-c6c37350d0d6"),
	},
	{
		WatchedDir: "activeTransfers/dataverseTransfer",
		Chain:      uuid.MustParse("10c00bc8-8fc2-419f-b593-cf5518695186"),
		Link:       uuid.MustParse("0af6b163-5455-4a76-978b-e35cc9ee445f"),
	},
}
