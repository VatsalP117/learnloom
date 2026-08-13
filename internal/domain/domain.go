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
	SearchIndexing bool           `json:"searchIndexing"`
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

type SourceRole string

const (
	SourceRoleOfficialPrimary SourceRole = "official_primary"
	SourceRoleResearch        SourceRole = "research"
	SourceRolePractitioner    SourceRole = "practitioner_explainer"
	SourceRoleReporting       SourceRole = "reporting_context"
	SourceRoleCounterweight   SourceRole = "counterweight"
)

type SourcePreference string

const (
	SourcePreferenceNeutral   SourcePreference = "neutral"
	SourcePreferencePreferred SourcePreference = "preferred"
	SourcePreferenceBlocked   SourcePreference = "blocked"
)

type SourceReviewMode string

const (
	SourceReviewAuto         SourceReviewMode = "auto"
	SourceReviewBeforeLesson SourceReviewMode = "review"
)

type SourceScoreComponents struct {
	SearchRank    int `json:"searchRank"`
	Relevance     int `json:"relevance"`
	Authority     int `json:"authority"`
	Primaryness   int `json:"primaryness"`
	Recency       int `json:"recency"`
	Usefulness    int `json:"usefulness"`
	Independence  int `json:"independence"`
	Accessibility int `json:"accessibility"`
	Counterweight int `json:"counterweight"`
	Negative      int `json:"negative"`
}

func (components SourceScoreComponents) Total() int {
	return components.SearchRank + components.Relevance + components.Authority +
		components.Primaryness + components.Recency + components.Usefulness +
		components.Independence + components.Accessibility +
		components.Counterweight + components.Negative
}

type SourceSpec struct {
	ID              string                `json:"id"`
	NewsletterID    string                `json:"newsletterId"`
	Origin          SourceOrigin          `json:"origin"`
	State           SourceState           `json:"state"`
	DisplayName     string                `json:"displayName"`
	InputURL        string                `json:"inputUrl"`
	CanonicalURL    string                `json:"canonicalUrl,omitempty"`
	Scope           SourceScope           `json:"scope"`
	Kind            SourceKind            `json:"kind,omitempty"`
	ItemLimit       int                   `json:"itemLimit"`
	DiscoveryReason string                `json:"discoveryReason,omitempty"`
	DiscoveryQuery  string                `json:"-"`
	RankScore       int                   `json:"-"`
	Role            SourceRole            `json:"role,omitempty"`
	RankingVersion  string                `json:"-"`
	ScoreComponents SourceScoreComponents `json:"-"`
	Preference      SourcePreference      `json:"preference"`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
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
	ID               string           `json:"id"`
	DisplayName      string           `json:"displayName"`
	CanonicalURL     string           `json:"canonicalUrl"`
	Origin           SourceOrigin     `json:"origin"`
	Scope            SourceScope      `json:"scope"`
	Kind             SourceKind       `json:"kind,omitempty"`
	State            SourceState      `json:"state"`
	Health           string           `json:"health"`
	DiscoveryReason  string           `json:"discoveryReason,omitempty"`
	Role             SourceRole       `json:"role,omitempty"`
	RankingVersion   string           `json:"rankingVersion,omitempty"`
	Preference       SourcePreference `json:"preference"`
	LastCheckedAt    *time.Time       `json:"lastCheckedAt,omitempty"`
	LastSuccessfulAt *time.Time       `json:"lastSuccessfulAt,omitempty"`
	Error            string           `json:"-"`
}

type Newsletter struct {
	ID                                 string             `json:"id"`
	OwnerAccountID                     string             `json:"-"`
	Name                               string             `json:"name"`
	Topic                              string             `json:"topic"`
	LearnerLevel                       string             `json:"learnerLevel"`
	LearnerGoal                        string             `json:"learnerGoal"`
	LessonMinutes                      int                `json:"lessonMinutes"`
	SourceMode                         SourceMode         `json:"sourceMode"`
	SourceReviewMode                   SourceReviewMode   `json:"sourceReviewMode"`
	SourceApprovedAt                   *time.Time         `json:"sourceApprovedAt,omitempty"`
	Sources                            []SourceDefinition `json:"sources"`
	ScheduleHour                       int                `json:"-"`
	ScheduleMinute                     int                `json:"-"`
	TimeZone                           string             `json:"timeZone"`
	RhythmMode                         RhythmMode         `json:"rhythmMode"`
	SelectedWeekdays                   []int              `json:"selectedWeekdays"`
	EffectiveRhythmMode                RhythmMode         `json:"effectiveRhythmMode"`
	AutoThrottleEnabled                bool               `json:"autoThrottleEnabled"`
	UnopenedLessonLimit                int                `json:"unopenedLessonLimit"`
	RhythmReason                       string             `json:"rhythmReason,omitempty"`
	RhythmThrottledAt                  *time.Time         `json:"rhythmThrottledAt,omitempty"`
	LessonPublicationDefault           PublicationState   `json:"lessonPublicationDefault"`
	LessonPublicationDefaultReviewedAt *time.Time         `json:"lessonPublicationDefaultReviewedAt,omitempty"`
	Active                             bool               `json:"active"`
	NextRunAt                          time.Time          `json:"nextRunAt"`
	EmailEnabled                       bool               `json:"emailEnabled"`
	EmailRecipients                    []string           `json:"emailRecipients"`
	AIExplorationEnabled               bool               `json:"aiExplorationEnabled"`
	PublicSlug                         string             `json:"publicSlug"`
	SiteVisible                        bool               `json:"siteVisible"`
	CreatedAt                          time.Time          `json:"createdAt"`
	UpdatedAt                          time.Time          `json:"updatedAt"`
}

type RhythmMode string

const (
	RhythmEvidenceLed      RhythmMode = "evidence_led"
	RhythmDaily            RhythmMode = "daily"
	RhythmSelectedWeekdays RhythmMode = "selected_weekdays"
	RhythmWeeklySynthesis  RhythmMode = "weekly_synthesis"
)

type IssueStatus string

const (
	IssueQueued           IssueStatus = "queued"
	IssueGenerating       IssueStatus = "generating"
	IssueAwaitingApproval IssueStatus = "awaiting_approval"
	IssueGenerated        IssueStatus = "generated"
	IssueFailed           IssueStatus = "failed"
	IssueDeferred         IssueStatus = "deferred"
	IssueCancelled        IssueStatus = "cancelled"
)

type IssueTrigger string

const (
	IssueScheduled IssueTrigger = "scheduled"
	IssueManual    IssueTrigger = "manual"
)

type PublicationState string

const (
	PublicationPublished PublicationState = "published"
	PublicationDraft     PublicationState = "draft"
	PublicationPrivate   PublicationState = "private"
)

type Issue struct {
	ID                     string           `json:"id"`
	NewsletterID           string           `json:"newsletterId"`
	Newsletter             Newsletter       `json:"newsletter"`
	Trigger                IssueTrigger     `json:"trigger"`
	ScheduledLocalDate     *string          `json:"scheduledLocalDate,omitempty"`
	Status                 IssueStatus      `json:"status"`
	Title                  string           `json:"title,omitempty"`
	GenerationID           string           `json:"generationId,omitempty"`
	ArtifactKey            string           `json:"-"`
	ArtifactSHA256         string           `json:"-"`
	ArtifactBytes          int              `json:"-"`
	Error                  string           `json:"-"`
	FailureCode            string           `json:"-"`
	FailureCategory        string           `json:"-"`
	FailureStage           string           `json:"-"`
	FailureRetryable       bool             `json:"-"`
	IncidentID             string           `json:"-"`
	PublicID               string           `json:"publicId,omitempty"`
	PublicSlug             string           `json:"publicSlug,omitempty"`
	PublicationState       PublicationState `json:"publicationState"`
	PublicationUpdatedAt   time.Time        `json:"publicationUpdatedAt"`
	FirstPublishReviewedAt *time.Time       `json:"firstPublishReviewedAt,omitempty"`
	PublishedAt            *time.Time       `json:"publishedAt,omitempty"`
	RequestedLessonType    LessonType       `json:"requestedLessonType,omitempty"`
	CreatedAt              time.Time        `json:"createdAt"`
	StartedAt              *time.Time       `json:"startedAt,omitempty"`
	CompletedAt            *time.Time       `json:"completedAt,omitempty"`
	Delivery               *DeliveryReceipt `json:"delivery,omitempty"`
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
	Error        string         `json:"-"`
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
	LessonType            LessonType `json:"lessonType,omitempty"`
	LearningObjective     string     `json:"learningObjective"`
	Prerequisites         []string   `json:"prerequisites"`
	Concepts              []string   `json:"concepts"`
	SuggestedNextConcepts []string   `json:"suggestedNextConcepts"`
	CentralMechanism      string     `json:"centralMechanism"`
	WorkedExample         string     `json:"workedExample"`
	Misconception         string     `json:"misconception"`
	PracticalExperiment   string     `json:"practicalExperiment"`
	ContinuityBridge      string     `json:"continuityBridge"`
}

type LessonType string

const (
	LessonFoundation  LessonType = "foundation"
	LessonUpdate      LessonType = "update"
	LessonDeepDive    LessonType = "deep_dive"
	LessonSynthesis   LessonType = "synthesis"
	LessonApplication LessonType = "application"
	LessonReview      LessonType = "review"
)

type EvidenceStatus string

const (
	EvidenceSourceBounded EvidenceStatus = "source_bounded"
)

type LearningConcept struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Role  string `json:"role"`
}

type EvidenceClaim struct {
	ID        string   `json:"id"`
	Text      string   `json:"text"`
	SourceIDs []string `json:"sourceIds"`
}

type ConceptChange struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Change string `json:"change"`
}

type RetrievalPrompt struct {
	ID                    string   `json:"id"`
	Prompt                string   `json:"prompt"`
	AnswerRubric          string   `json:"answerRubric"`
	CorrectiveExplanation string   `json:"correctiveExplanation"`
	ConceptIDs            []string `json:"conceptIds"`
}

type LearningContract struct {
	Version               int               `json:"version"`
	LessonType            LessonType        `json:"lessonType"`
	EvidenceStatus        EvidenceStatus    `json:"evidenceStatus"`
	SelectionRationale    string            `json:"selectionRationale"`
	LearningObjective     string            `json:"learningObjective"`
	ContinuityBridge      string            `json:"continuityBridge"`
	Concepts              []LearningConcept `json:"concepts"`
	ConceptChanges        []ConceptChange   `json:"conceptChanges"`
	Misconception         string            `json:"misconception"`
	Claims                []EvidenceClaim   `json:"claims"`
	Limitations           []EvidenceClaim   `json:"limitations"`
	Retrieval             []RetrievalPrompt `json:"retrieval"`
	SuggestedNextConcepts []string          `json:"suggestedNextConcepts"`
	Application           string            `json:"application"`
}

type LearnerConceptProgress struct {
	Label              string `json:"label"`
	Role               string `json:"role"`
	ExposureCount      int    `json:"exposureCount"`
	CompletedCount     int    `json:"completedCount"`
	ReviewAttemptCount int    `json:"reviewAttemptCount"`
	ConfidenceScore    int    `json:"confidenceScore"`
}

type LearnerState struct {
	Concepts         []LearnerConceptProgress `json:"concepts"`
	Difficulty       string                   `json:"difficulty,omitempty"`
	Relevance        string                   `json:"relevance,omitempty"`
	RecallConfidence string                   `json:"recallConfidence,omitempty"`
	OpenQuestions    []string                 `json:"openQuestions,omitempty"`
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
	LessonType  LessonType        `json:"lessonType"`
	GeneratedAt time.Time         `json:"generatedAt"`
	Model       string            `json:"model"`
	Curation    Curation          `json:"curation"`
	Blueprint   LearningBlueprint `json:"blueprint"`
	Learning    LearningContract  `json:"learning"`
	Lesson      string            `json:"lesson"`
	Critique    string            `json:"critique"`
	Practice    string            `json:"practice"`
	Exploration *string           `json:"exploration"`
	Quality     QualityReport     `json:"quality"`
	Sources     []SourceItem      `json:"sources"`
}

type LearningHistoryEntry struct {
	Date                  string            `json:"date"`
	GeneratedAt           time.Time         `json:"generatedAt"`
	LessonType            LessonType        `json:"lessonType,omitempty"`
	SourceTitles          []string          `json:"sourceTitles"`
	SourceURLs            []string          `json:"sourceUrls,omitempty"`
	LessonSummary         string            `json:"lessonSummary"`
	RecallQuestions       []string          `json:"recallQuestions"`
	RetrievalPrompts      []RetrievalPrompt `json:"retrievalPrompts,omitempty"`
	ConceptStates         []LearningConcept `json:"conceptStates,omitempty"`
	SuggestedNextConcepts []string          `json:"suggestedNextConcepts,omitempty"`
	LearningObjective     string            `json:"learningObjective"`
	Concepts              []string          `json:"concepts"`
}

type DossierArtifact struct {
	Dossier  Dossier `json:"dossier"`
	Markdown string  `json:"markdown"`
	HTML     string  `json:"html"`
}
