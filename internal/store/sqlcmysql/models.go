// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package sqlcmysql

import (
	"database/sql"
	"time"

	uuid "github.com/google/uuid"
)

type Access struct {
	ID         int32
	SIPID      uuid.UUID
	Resource   string
	Target     string
	Status     string
	Statuscode sql.NullInt16
	Exitcode   sql.NullInt16
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Agent struct {
	ID                   int32
	Agentidentifiertype  sql.NullString
	Agentidentifiervalue sql.NullString
	Agentname            sql.NullString
	Agenttype            string
}

type Archivesspacedipobjectresourcepairing struct {
	ID         int32
	Dipuuid    uuid.UUID
	Fileuuid   uuid.UUID
	Resourceid string
}

type AuthGroup struct {
	ID   int32
	Name string
}

type AuthGroupPermission struct {
	ID           int32
	GroupID      int32
	PermissionID int32
}

type AuthPermission struct {
	ID            int32
	Name          string
	ContentTypeID int32
	Codename      string
}

type AuthUser struct {
	ID          int32
	Password    string
	LastLogin   sql.NullTime
	IsSuperuser bool
	Username    string
	FirstName   string
	LastName    string
	Email       string
	IsStaff     bool
	IsActive    bool
	DateJoined  time.Time
}

type AuthUserGroup struct {
	ID      int32
	UserID  int32
	GroupID int32
}

type AuthUserUserPermission struct {
	ID           int32
	UserID       int32
	PermissionID int32
}

type Dashboardsetting struct {
	ID           int32
	Name         string
	Value        string
	Lastmodified time.Time
	Scope        string
}

type Derivation struct {
	ID               int32
	Derivedfileuuid  uuid.UUID
	Relatedeventuuid uuid.UUID
	Sourcefileuuid   uuid.UUID
}

type DirectoriesIdentifier struct {
	ID           int32
	DirectoryID  string
	IdentifierID int32
}

type Directory struct {
	Directoryuuid    uuid.UUID
	Originallocation []byte
	Currentlocation  sql.NullString
	Enteredsystem    time.Time
	SIPID            uuid.UUID
	Transferuuid     uuid.UUID
}

type DjangoContentType struct {
	ID       int32
	AppLabel string
	Model    string
}

type DjangoMigration struct {
	ID      int32
	App     string
	Name    string
	Applied time.Time
}

type DjangoSession struct {
	SessionKey  string
	SessionData string
	ExpireDate  time.Time
}

type Dublincore struct {
	ID                          int32
	Metadataappliestoidentifier sql.NullString
	Title                       string
	Ispartof                    string
	Creator                     string
	Subject                     string
	Description                 string
	Publisher                   string
	Contributor                 string
	Date                        string
	Type                        string
	Format                      string
	Identifier                  string
	Source                      string
	Relation                    string
	Language                    string
	Coverage                    string
	Rights                      string
	Metadataappliestotype       string
	Status                      string
}

type Event struct {
	ID                     int32
	Eventidentifieruuid    uuid.UUID
	Eventtype              string
	Eventdatetime          time.Time
	Eventdetail            string
	Eventoutcome           string
	Eventoutcomedetailnote string
	Fileuuid               uuid.UUID
}

type EventsAgent struct {
	ID      int32
	EventID int32
	AgentID int32
}

type File struct {
	Fileuuid         uuid.UUID
	Originallocation []byte
	Currentlocation  sql.NullString
	Filegrpuse       string
	Filegrpuuid      uuid.UUID
	Checksum         string
	Filesize         sql.NullInt64
	Label            string
	Enteredsystem    time.Time
	Removedtime      sql.NullTime
	SIPID            uuid.UUID
	Transferuuid     uuid.UUID
	Checksumtype     string
	Modificationtime sql.NullTime
}

type FilesIdentifier struct {
	ID           int32
	FileID       string
	IdentifierID int32
}

type Filesid struct {
	ID                 int32
	Formatname         string
	Formatversion      string
	Formatregistryname string
	Formatregistrykey  string
	Fileuuid           uuid.UUID
}

type Filesidentifiedid struct {
	ID       int32
	Fileuuid uuid.UUID
	Fileid   string
}

type FprFormat struct {
	ID          int32
	UUID        uuid.UUID
	Description string
	Slug        string
	GroupID     sql.NullString
}

type FprFormatgroup struct {
	ID          int32
	UUID        uuid.UUID
	Description string
	Slug        string
}

type FprFormatversion struct {
	ID                 int32
	Enabled            bool
	Lastmodified       time.Time
	UUID               uuid.UUID
	Version            sql.NullString
	PronomID           sql.NullString
	Description        sql.NullString
	AccessFormat       bool
	PreservationFormat bool
	Slug               string
	FormatID           sql.NullString
	ReplacesID         sql.NullString
}

type FprFpcommand struct {
	ID                    int32
	Enabled               bool
	Lastmodified          time.Time
	UUID                  uuid.UUID
	Description           string
	Command               string
	ScriptType            string
	OutputLocation        sql.NullString
	CommandUsage          string
	EventDetailCommandID  sql.NullString
	OutputFormatID        sql.NullString
	ReplacesID            sql.NullString
	ToolID                sql.NullString
	VerificationCommandID sql.NullString
}

type FprFprule struct {
	ID            int32
	Enabled       bool
	Lastmodified  time.Time
	UUID          uuid.UUID
	Purpose       string
	CountAttempts int32
	CountOkay     int32
	CountNotOkay  int32
	CommandID     string
	FormatID      string
	ReplacesID    sql.NullString
}

type FprFptool struct {
	ID          int32
	UUID        uuid.UUID
	Description string
	Version     string
	Enabled     bool
	Slug        string
}

type FprIdcommand struct {
	ID           int32
	Enabled      bool
	Lastmodified time.Time
	UUID         uuid.UUID
	Description  string
	Config       string
	Script       string
	ScriptType   string
	ReplacesID   sql.NullString
	ToolID       sql.NullString
}

type FprIdrule struct {
	ID            int32
	Enabled       bool
	Lastmodified  time.Time
	UUID          uuid.UUID
	CommandOutput string
	CommandID     string
	FormatID      string
	ReplacesID    sql.NullString
}

type FprIdtool struct {
	ID          int32
	UUID        uuid.UUID
	Description string
	Version     string
	Enabled     bool
	Slug        string
}

type Identifier struct {
	ID    int32
	Type  sql.NullString
	Value sql.NullString
}

type Job struct {
	ID                uuid.UUID
	Type              string
	CreatedAt         time.Time
	Createdtimedec    string
	Directory         string
	SIPID             uuid.UUID
	Unittype          string
	Currentstep       int32
	Microservicegroup string
	Hidden            bool
	Subjobof          string
	LinkID            uuid.NullUUID
}

type MainArchivesspacedigitalobject struct {
	ID         int32
	Resourceid string
	Label      string
	Title      string
	Started    bool
	Remoteid   string
	SipID      sql.NullString
}

type MainFpcommandoutput struct {
	ID       int32
	Content  sql.NullString
	Fileuuid uuid.UUID
	Ruleuuid uuid.UUID
}

type MainLevelofdescription struct {
	ID        string
	Name      string
	Sortorder int32
}

type MainSiparrange struct {
	ID                 int32
	OriginalPath       sql.NullString
	ArrangePath        []byte
	FileUuid           uuid.UUID
	TransferUuid       uuid.UUID
	SipCreated         bool
	AipCreated         bool
	LevelOfDescription string
	SipID              sql.NullString
}

type MainSiparrangeaccessmapping struct {
	ID          int32
	ArrangePath string
	System      string
	Identifier  string
}

type MainUserprofile struct {
	ID           int32
	AgentID      int32
	UserID       int32
	SystemEmails bool
}

type Metadataappliestotype struct {
	ID           string
	Description  string
	Replaces     sql.NullString
	Lastmodified time.Time
}

type Report struct {
	ID             int32
	Unittype       string
	Unitname       string
	Unitidentifier string
	Content        string
	Created        time.Time
}

type Rightsstatement struct {
	ID                             int32
	Metadataappliestoidentifier    string
	Rightsstatementidentifiertype  string
	Rightsstatementidentifiervalue string
	Fkagent                        int32
	Rightsbasis                    string
	Metadataappliestotype          string
	Status                         string
}

type Rightsstatementcopyright struct {
	ID                               int32
	Copyrightstatus                  string
	Copyrightjurisdiction            string
	Copyrightstatusdeterminationdate sql.NullString
	Copyrightapplicablestartdate     sql.NullString
	Copyrightapplicableenddate       sql.NullString
	Copyrightapplicableenddateopen   bool
	Fkrightsstatement                int32
}

type Rightsstatementcopyrightdocumentationidentifier struct {
	ID                                    int32
	Copyrightdocumentationidentifiertype  string
	Copyrightdocumentationidentifiervalue string
	Copyrightdocumentationidentifierrole  sql.NullString
	Fkrightsstatementcopyrightinformation int32
}

type Rightsstatementcopyrightnote struct {
	ID                                    int32
	Copyrightnote                         string
	Fkrightsstatementcopyrightinformation int32
}

type Rightsstatementlicense struct {
	ID                           int32
	Licenseterms                 sql.NullString
	Licenseapplicablestartdate   sql.NullString
	Licenseapplicableenddate     sql.NullString
	Licenseapplicableenddateopen bool
	Fkrightsstatement            int32
}

type Rightsstatementlicensedocumentationidentifier struct {
	ID                                  int32
	Licensedocumentationidentifiertype  string
	Licensedocumentationidentifiervalue string
	Licensedocumentationidentifierrole  sql.NullString
	Fkrightsstatementlicense            int32
}

type Rightsstatementlicensenote struct {
	ID                       int32
	Licensenote              string
	Fkrightsstatementlicense int32
}

type Rightsstatementlinkingagentidentifier struct {
	ID                          int32
	Linkingagentidentifiertype  string
	Linkingagentidentifiervalue string
	Fkrightsstatement           int32
}

type Rightsstatementotherrightsdocumentationidentifier struct {
	ID                                      int32
	Otherrightsdocumentationidentifiertype  string
	Otherrightsdocumentationidentifiervalue string
	Otherrightsdocumentationidentifierrole  sql.NullString
	Fkrightsstatementotherrightsinformation int32
}

type Rightsstatementotherrightsinformation struct {
	ID                               int32
	Otherrightsbasis                 string
	Otherrightsapplicablestartdate   sql.NullString
	Otherrightsapplicableenddate     sql.NullString
	Otherrightsapplicableenddateopen bool
	Fkrightsstatement                int32
}

type Rightsstatementotherrightsnote struct {
	ID                                      int32
	Otherrightsnote                         string
	Fkrightsstatementotherrightsinformation int32
}

type Rightsstatementrightsgranted struct {
	ID                int32
	Act               string
	Startdate         sql.NullString
	Enddate           sql.NullString
	Enddateopen       bool
	Fkrightsstatement int32
}

type Rightsstatementrightsgrantednote struct {
	ID                             int32
	Rightsgrantednote              string
	Fkrightsstatementrightsgranted int32
}

type Rightsstatementrightsgrantedrestriction struct {
	ID                             int32
	Restriction                    string
	Fkrightsstatementrightsgranted int32
}

type Rightsstatementstatutedocumentationidentifier struct {
	ID                                  int32
	Statutedocumentationidentifiertype  string
	Statutedocumentationidentifiervalue string
	Statutedocumentationidentifierrole  sql.NullString
	Fkrightsstatementstatuteinformation int32
}

type Rightsstatementstatuteinformation struct {
	ID                                  int32
	Statutejurisdiction                 string
	Statutecitation                     string
	Statuteinformationdeterminationdate sql.NullString
	Statuteapplicablestartdate          sql.NullString
	Statuteapplicableenddate            sql.NullString
	Statuteapplicableenddateopen        bool
	Fkrightsstatement                   int32
}

type Rightsstatementstatuteinformationnote struct {
	ID                                  int32
	Statutenote                         string
	Fkrightsstatementstatuteinformation int32
}

type Sip struct {
	SIPID       uuid.UUID
	CreatedAt   time.Time
	Currentpath sql.NullString
	Hidden      bool
	Aipfilename sql.NullString
	Siptype     string
	Diruuids    bool
	CompletedAt sql.NullTime
	Status      uint16
}

type SipsIdentifier struct {
	ID           int32
	SipID        string
	IdentifierID int32
}

type Task struct {
	Taskuuid  uuid.UUID
	CreatedAt time.Time
	Fileuuid  uuid.UUID
	Filename  string
	Exec      string
	Arguments string
	Starttime sql.NullTime
	Endtime   sql.NullTime
	Client    string
	Stdout    string
	Stderror  string
	Exitcode  sql.NullInt64
	ID        uuid.UUID
}

type TastypieApiaccess struct {
	ID            int32
	Identifier    string
	Url           string
	RequestMethod string
	Accessed      uint32
}

type TastypieApikey struct {
	ID      int32
	Key     string
	Created time.Time
	UserID  int32
}

type Taxonomy struct {
	ID        string
	CreatedAt sql.NullTime
	Name      string
	Type      string
}

type Taxonomyterm struct {
	ID           string
	CreatedAt    sql.NullTime
	Term         string
	Taxonomyuuid uuid.UUID
}

type Transfer struct {
	Transferuuid               uuid.UUID
	Currentlocation            string
	Type                       string
	Accessionid                string
	Sourceofacquisition        string
	Typeoftransfer             string
	Description                string
	Notes                      string
	Hidden                     bool
	Transfermetadatasetrowuuid uuid.NullUUID
	Diruuids                   bool
	AccessSystemID             string
	CompletedAt                sql.NullTime
	Status                     uint16
}

type Transfermetadatafield struct {
	ID                 string
	CreatedAt          sql.NullTime
	Fieldlabel         string
	Fieldname          string
	Fieldtype          string
	Sortorder          int32
	Optiontaxonomyuuid uuid.UUID
}

type Transfermetadatafieldvalue struct {
	ID         string
	CreatedAt  time.Time
	Fieldvalue string
	Fielduuid  uuid.UUID
	Setuuid    uuid.UUID
}

type Transfermetadataset struct {
	ID              string
	CreatedAt       time.Time
	Createdbyuserid int32
}

type Unitvariable struct {
	ID            uuid.UUID
	Unittype      sql.NullString
	Unituuid      uuid.UUID
	Variable      sql.NullString
	Variablevalue sql.NullString
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LinkID        uuid.NullUUID
}
