package domain

import "time"

type AccountStatus string

const (
	AccountActive    AccountStatus = "active"
	AccountSuspended AccountStatus = "suspended"
	AccountDeleted   AccountStatus = "deleted"
)

type Account struct {
	ID           string        `json:"id"`
	ClerkUserID  string        `json:"-"`
	PrimaryEmail string        `json:"primaryEmail,omitempty"`
	Status       AccountStatus `json:"status"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	DeletedAt    *time.Time    `json:"deletedAt,omitempty"`
}

type SiteVisibility string

const (
	SitePrivate SiteVisibility = "private"
	SitePublic  SiteVisibility = "public"
)

type PersonalSite struct {
	ID             string         `json:"id"`
	OwnerAccountID string         `json:"-"`
	Username       string         `json:"username"`
	DisplayName    string         `json:"displayName"`
	Description    string         `json:"description"`
	Visibility     SiteVisibility `json:"visibility"`
	ClaimedAt      time.Time      `json:"claimedAt"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type SourceDefinition struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Limit int    `json:"limit"`
}

type SourceMode string

const (
	SourceModeDiscovered SourceMode = "discovered"
	SourceModeProvided   SourceMode = "provided"
	SourceModeHybrid     SourceMode = "hybrid"
)

type SourceOrigin string

const (
	SourceOriginProvided   SourceOrigin = "provided"
	SourceOriginDiscovered SourceOrigin = "discovered"
)

type SourceScope string

const (
	SourceScopeExact    SourceScope = "exact"
	SourceScopeFeed     SourceScope = "feed"
	SourceScopeSite     SourceScope = "site"
	SourceScopeDocument SourceScope = "document"
)

type SourceKind string

const (
	SourceKindRSS      SourceKind = "rss"
	SourceKindAtom     SourceKind = "atom"
	SourceKindJSONFeed SourceKind = "json_feed"
	SourceKindHTML     SourceKind = "html"
	SourceKindText     SourceKind = "text"
	SourceKindPDF      SourceKind = "pdf"
)

type SourceState string

const (
	SourceStateCandidate SourceState = "candidate"
	SourceStateActive    SourceState = "active"
	SourceStateUnhealthy SourceState = "unhealthy"
	SourceStateRejected  SourceState = "rejected"
	SourceStateDisabled  SourceState = "disabled"
)

type SourceSpec struct {
	ID              string       `json:"id"`
	NewsletterID    string       `json:"newsletterId"`
	Origin          SourceOrigin `json:"origin"`
	State           SourceState  `json:"state"`
	DisplayName     string       `json:"displayName"`
	InputURL        string       `json:"inputUrl"`
	CanonicalURL    string       `json:"canonicalUrl,omitempty"`
	Scope           SourceScope  `json:"scope"`
	Kind            SourceKind   `json:"kind,omitempty"`
	ItemLimit       int          `json:"itemLimit"`
	DiscoveryReason string       `json:"discoveryReason,omitempty"`
	DiscoveryQuery  string       `json:"-"`
	RankScore       int          `json:"-"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
}

type SourceEndpoint struct {
	ID                  string     `json:"id"`
	SourceSpecID        string     `json:"sourceSpecId"`
	EndpointURL         string     `json:"endpointUrl"`
	CanonicalURL        string     `json:"canonicalUrl"`
	Kind                SourceKind `json:"kind"`
	ETag                string     `json:"-"`
	LastModified        string     `json:"-"`
	LastHTTPStatus      int        `json:"-"`
	Health              string     `json:"health"`
	ConsecutiveFailures int        `json:"-"`
	LastCheckedAt       *time.Time `json:"lastCheckedAt,omitempty"`
	LastSuccessAt       *time.Time `json:"lastSuccessAt,omitempty"`
	LastChangedAt       *time.Time `json:"lastChangedAt,omitempty"`
	LastError           string     `json:"-"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type SourceSnapshot struct {
	ID               string     `json:"id"`
	SourceEndpointID string     `json:"sourceEndpointId"`
	ItemKey          string     `json:"-"`
	Title            string     `json:"title"`
	CanonicalURL     string     `json:"canonicalUrl"`
	Author           string     `json:"author,omitempty"`
	PublishedAt      *time.Time `json:"publishedAt,omitempty"`
	Content          string     `json:"-"`
	ContentSource    string     `json:"contentSource"`
	ContentSHA256    string     `json:"-"`
	Metadata         string     `json:"-"`
	FetchedAt        time.Time  `json:"fetchedAt"`
}

type IssueSource struct {
	IssueID          string    `json:"issueId"`
	SourceSnapshotID string    `json:"sourceSnapshotId"`
	Position         int       `json:"position"`
	CreatedAt        time.Time `json:"createdAt"`
}

type DiscoveryRun struct {
	ID                  string     `json:"id"`
	NewsletterID        string     `json:"newsletterId"`
	IssueID             string     `json:"issueId,omitempty"`
	Reason              string     `json:"reason"`
	State               string     `json:"state"`
	QueryBundle         string     `json:"-"`
	ReturnedCandidates  int        `json:"returnedCandidates"`
	RejectedCandidates  int        `json:"rejectedCandidates"`
	ResolvedCandidates  int        `json:"resolvedCandidates"`
	ActivatedCandidates int        `json:"activatedCandidates"`
	Error               string     `json:"-"`
	StartedAt           *time.Time `json:"startedAt,omitempty"`
	CompletedAt         *time.Time `json:"completedAt,omitempty"`
}

type SourceSummary struct {
	Provided       int        `json:"provided"`
	Discovered     int        `json:"discovered"`
	Healthy        int        `json:"healthy"`
	NeedsAttention int        `json:"needsAttention"`
	LastCheckedAt  *time.Time `json:"lastCheckedAt,omitempty"`
}

type SourceCatalogItem struct {
	ID               string       `json:"id"`
	DisplayName      string       `json:"displayName"`
	CanonicalURL     string       `json:"canonicalUrl"`
	Origin           SourceOrigin `json:"origin"`
	Scope            SourceScope  `json:"scope"`
	Kind             SourceKind   `json:"kind,omitempty"`
	State            SourceState  `json:"state"`
	Health           string       `json:"health"`
	DiscoveryReason  string       `json:"discoveryReason,omitempty"`
	LastCheckedAt    *time.Time   `json:"lastCheckedAt,omitempty"`
	LastSuccessfulAt *time.Time   `json:"lastSuccessfulAt,omitempty"`
	Error            string       `json:"error,omitempty"`
}

type Newsletter struct {
	ID                   string             `json:"id"`
	OwnerAccountID       string             `json:"-"`
	Name                 string             `json:"name"`
	Topic                string             `json:"topic"`
	LearnerLevel         string             `json:"learnerLevel"`
	LearnerGoal          string             `json:"learnerGoal"`
	LessonMinutes        int                `json:"lessonMinutes"`
	SourceMode           SourceMode         `json:"sourceMode"`
	Sources              []SourceDefinition `json:"sources"`
	ScheduleHour         int                `json:"-"`
	ScheduleMinute       int                `json:"-"`
	TimeZone             string             `json:"timeZone"`
	Active               bool               `json:"active"`
	NextRunAt            time.Time          `json:"nextRunAt"`
	EmailEnabled         bool               `json:"emailEnabled"`
	EmailRecipients      []string           `json:"emailRecipients"`
	AIExplorationEnabled bool               `json:"aiExplorationEnabled"`
	PublicSlug           string             `json:"publicSlug"`
	SiteVisible          bool               `json:"siteVisible"`
	CreatedAt            time.Time          `json:"createdAt"`
	UpdatedAt            time.Time          `json:"updatedAt"`
}

type IssueStatus string

const (
	IssueQueued     IssueStatus = "queued"
	IssueGenerating IssueStatus = "generating"
	IssueGenerated  IssueStatus = "generated"
	IssueFailed     IssueStatus = "failed"
	IssueCancelled  IssueStatus = "cancelled"
)

type IssueTrigger string

const (
	IssueScheduled IssueTrigger = "scheduled"
	IssueManual    IssueTrigger = "manual"
)

type PublicationState string

const (
	PublicationPublished PublicationState = "published"
	PublicationHidden    PublicationState = "hidden"
)

type Issue struct {
	ID                 string           `json:"id"`
	NewsletterID       string           `json:"newsletterId"`
	Newsletter         Newsletter       `json:"newsletter"`
	Trigger            IssueTrigger     `json:"trigger"`
	ScheduledLocalDate *string          `json:"scheduledLocalDate,omitempty"`
	Status             IssueStatus      `json:"status"`
	Title              string           `json:"title,omitempty"`
	GenerationID       string           `json:"generationId,omitempty"`
	ArtifactKey        string           `json:"-"`
	Error              string           `json:"error,omitempty"`
	PublicID           string           `json:"publicId,omitempty"`
	PublicSlug         string           `json:"publicSlug,omitempty"`
	PublicationState   PublicationState `json:"publicationState"`
	CreatedAt          time.Time        `json:"createdAt"`
	StartedAt          *time.Time       `json:"startedAt,omitempty"`
	CompletedAt        *time.Time       `json:"completedAt,omitempty"`
	Delivery           *DeliveryReceipt `json:"delivery,omitempty"`
}

type DeliveryStatus string

const (
	DeliveryPending    DeliveryStatus = "pending"
	DeliveryDelivering DeliveryStatus = "delivering"
	DeliveryDelivered  DeliveryStatus = "delivered"
	DeliveryFailed     DeliveryStatus = "failed"
	DeliveryCancelled  DeliveryStatus = "cancelled"
	DeliveryUnknown    DeliveryStatus = "unknown"
)

type DeliveryReceipt struct {
	IssueID      string         `json:"issueId"`
	Status       DeliveryStatus `json:"status"`
	AttemptCount int            `json:"attemptCount"`
	ExternalID   string         `json:"externalId,omitempty"`
	Error        string         `json:"error,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	StartedAt    *time.Time     `json:"startedAt,omitempty"`
	CompletedAt  *time.Time     `json:"completedAt,omitempty"`
	NextAttempt  *time.Time     `json:"nextAttemptAt,omitempty"`
}

type SourceItem struct {
	SourceID        string     `json:"sourceId"`
	OriginalID      string     `json:"originalSourceId,omitempty"`
	Source          string     `json:"source"`
	Title           string     `json:"title"`
	URL             string     `json:"url"`
	CanonicalURL    string     `json:"canonicalUrl"`
	Summary         string     `json:"summary"`
	PublishedAt     *time.Time `json:"publishedAt,omitempty"`
	ContentSource   string     `json:"contentSource"`
	Author          string     `json:"author,omitempty"`
	EnrichmentError string     `json:"enrichmentError,omitempty"`
}

type Curation struct {
	Theme             string   `json:"theme"`
	Rationale         string   `json:"rationale"`
	SelectedSourceIDs []string `json:"selectedSourceIds"`
}

type LearningBlueprint struct {
	LearningObjective   string   `json:"learningObjective"`
	Prerequisites       []string `json:"prerequisites"`
	CentralMechanism    string   `json:"centralMechanism"`
	WorkedExample       string   `json:"workedExample"`
	Misconception       string   `json:"misconception"`
	PracticalExperiment string   `json:"practicalExperiment"`
	ContinuityBridge    string   `json:"continuityBridge"`
}

type QualityReport struct {
	Version     int             `json:"version"`
	Score       int             `json:"score"`
	Checks      map[string]bool `json:"checks"`
	Metrics     map[string]int  `json:"metrics"`
	EditorNotes []string        `json:"editorNotes"`
}

type Dossier struct {
	Version     int               `json:"version"`
	ProfileID   string            `json:"profileId"`
	Date        string            `json:"date"`
	Title       string            `json:"title"`
	GeneratedAt time.Time         `json:"generatedAt"`
	Model       string            `json:"model"`
	Curation    Curation          `json:"curation"`
	Blueprint   LearningBlueprint `json:"blueprint"`
	Lesson      string            `json:"lesson"`
	Critique    string            `json:"critique"`
	Practice    string            `json:"practice"`
	Exploration *string           `json:"exploration"`
	Quality     QualityReport     `json:"quality"`
	Sources     []SourceItem      `json:"sources"`
}

type LearningHistoryEntry struct {
	Date              string    `json:"date"`
	GeneratedAt       time.Time `json:"generatedAt"`
	SourceTitles      []string  `json:"sourceTitles"`
	LessonSummary     string    `json:"lessonSummary"`
	RecallQuestions   []string  `json:"recallQuestions"`
	LearningObjective string    `json:"learningObjective"`
	Concepts          []string  `json:"concepts"`
}

type DossierArtifact struct {
	Dossier  Dossier `json:"dossier"`
	Markdown string  `json:"markdown"`
	HTML     string  `json:"html"`
}
