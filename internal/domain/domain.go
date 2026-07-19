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

type Newsletter struct {
	ID                   string             `json:"id"`
	OwnerAccountID       string             `json:"-"`
	Name                 string             `json:"name"`
	Topic                string             `json:"topic"`
	LearnerLevel         string             `json:"learnerLevel"`
	LearnerGoal          string             `json:"learnerGoal"`
	LessonMinutes        int                `json:"lessonMinutes"`
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
