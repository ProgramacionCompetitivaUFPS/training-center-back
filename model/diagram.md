erDiagram


%% ===== Main Entities =====

User {
  string id PK
  string email UK "nullable after deactivation"
  string password "hashed"
  string name
  string country "required"
  string city "required"
  string institution "required"
  string nickname UK "lowercase, anonymized after deactivation"
  enum role "ADMIN | COACH | CONTESTANT"
  enum status "ACTIVE | DEACTIVATED"
  timestamp deactivatedAt "nullable"
  timestamp createdAt
  timestamp updatedAt "nullable"
}

Group {
  string id PK
  string name
  string description
  boolean isGlobal "default false"
  timestamp createdAt
  timestamp updatedAt
}

Problem {
  string id PK
  string slug UK "user-provided, 3-70 chars, lowercase"
  string title
  string statement "LaTeX format"
  integer timeLimit "milliseconds, default for all languages"
  integer memoryLimit "MiB, default for all languages"
  json languageOverrides "language-specific limits array"
  string[] tags "optional, from predefined list"
  enum status "DRAFT | PUBLISHED"
  enum accessibility "PUBLIC | PRIVATE"
  string authorId FK
  string[] modifierIds FK "array of User IDs"
  string testCasesFileKey "nullable"
  string[] solutionFileKeys "array"
  string checkerFileKey "nullable"
  string validatorFileKey "nullable"
  timestamp problemJudgingUpdatedAt "nullable"
  timestamp createdAt
  timestamp updatedAt
}

Contest {
  string id PK
  string name
  string description
  string group_id FK
  string ownerId FK
  timestamp startTime
  timestamp endTime
  integer penalty "minutes, max 1440"
  integer freezeMinutes "nullable, default 60"
  boolean locked "default false"
  boolean enablePostContest "default false"
  timestamp createdAt
  timestamp updatedAt
}

Contest_Problem {
  string id PK
  string contest_id FK
  string problem_id FK
  integer position "order in contest"
}

Submission {
  string id PK
  string problem_id FK
  string contest_id FK "nullable"
  string contestant_id FK
  enum status "PENDING | RUNNING | ACCEPTED | WRONG_ANSWER | TLE | MLE | RE | CE | PE"
  string language "cpp20, java17, python310"
  string compiler "g++, javac, pypy3"
  string filePath "storage path/key"
  string fileHash "SHA256"
  integer fileSize "bytes"
  timestamp submittedAt "captured immediately on request"
  timestamp judgedAt "nullable, set when judging completes"
  integer executionTime "milliseconds, max across all cases"
  integer memoryUsed "KB, max across all cases"
  string compilationLog "nullable, only for CE, max 10KB"
}

Material {
  string id PK
  string group_id FK
  string author_id FK
  string title "1-200 chars"
  string content "Markdown with embedded URLs, max 50000 chars"
  string[] tags "user-defined, lowercase + numbers + hyphens + underscores"
  enum status "DRAFT | PUBLISHED"
  boolean pinned "default false"
  timestamp pinnedAt "nullable"
  timestamp publishedAt "nullable"
  timestamp createdAt
  timestamp updatedAt
}


%% ===== Group Membership =====

GroupMember {
  string id PK
  string user_id FK
  string group_id FK
  enum memberRole "MEMBER | LEAD"
  timestamp joinedAt
}


%% ===== Contest Registration & Standing (NoSQL) =====
%% Note: In implementation, these are stored in NoSQL collections:
%% - contest_{contestId}_standings (active)
%% - contest_{contestId}_standings_final (snapshot)

ContestParticipant {
  string contestantId PK
  timestamp registeredAt
  integer problemsSolved "0 on registration"
  integer penalty "total penalty in minutes"
  json problems "array of problem attempts"
}


%% ===== Authentication & Security Entities =====

PasswordRecoveryRequest {
  string id PK
  string userId FK
  string email
  string verificationCode "6-digit"
  timestamp expiresAt "15 minutes"
  enum status "PENDING | COMPLETED | EXPIRED"
  timestamp createdAt
}

EmailChangeRequest {
  string id PK
  string userId FK
  string newEmail
  string verificationCode "6-digit"
  timestamp expiresAt "15 minutes"
  enum status "PENDING | COMPLETED | EXPIRED | CANCELLED"
  timestamp createdAt
}

DeactivationRequest {
  string id PK
  string userId FK
  string verificationCode "6-digit"
  timestamp expiresAt "15 minutes"
  integer attempts "max 5"
  timestamp blockedUntil "nullable, 1-hour block"
  enum status "PENDING | CONFIRMED | EXPIRED | BLOCKED"
  timestamp createdAt
}

DeactivationAuditLog {
  string id PK
  string userId FK
  string originalEmail
  string originalNickname
  timestamp occurredAt
  string ip "nullable"
  string userAgent "nullable"
  enum deactivationType "SELF | ADMIN"
  string adminId FK "nullable, only for ADMIN type"
}

RecoveryRateLimit {
  string email PK
  integer requestCount
  timestamp windowStart "1-hour window"
}

PasswordUpdateAttempt {
  string userId PK
  integer failedAttempts
  timestamp lastAttemptAt
  timestamp cooldownUntil "nullable"
}


%% ===== User Role Relationships (pseudo-entities for visualization) =====
%% Note: These are the user roles (User.role field)

Admin
Coach
Contestant

User ||--o{ Admin : "is (role=ADMIN)"
User ||--o{ Coach : "is (role=COACH)"
User ||--o{ Contestant : "is (role=CONTESTANT)"

%% Note: Admin has implicit permissions on ALL groups
%% without requiring registration in GroupMember (system-level permissions)


%% ===== Group Membership Relationships =====
%% Note: Members and Leads are represented via GroupMember.memberRole

User ||--o{ GroupMember : has
GroupMember }o--|| Group : belongs_to

Group ||--o{ Material : has
Group ||--o{ Contest : organizes

%% ===== Material Relationships =====

Material }o--|| User : authored_by
Material }o--|| Group : belongs_to


%% ===== Problem Relationships =====

Problem }o--|| User : authored_by
Problem ||--o{ Submission : receives
Problem ||--o{ Contest_Problem : included_in


%% ===== Contest Relationships =====

Contest ||--o{ Contest_Problem : includes
Contest }o--|| User : owned_by
Contest }o--|| Group : belongs_to
Contest ||--o{ ContestParticipant : has_participants


%% ===== Submission Relationships =====

Submission }o--|| User : submitted_by
Submission }o--|| Problem : solves
Submission }o--o| Contest : may_belong_to


%% ===== User Security Relationships =====

User ||--o{ PasswordRecoveryRequest : requests
User ||--o{ EmailChangeRequest : requests
User ||--o{ DeactivationRequest : requests
User ||--o{ DeactivationAuditLog : has
User ||--o{ PasswordUpdateAttempt : has

