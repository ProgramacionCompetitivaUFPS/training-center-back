erDiagram


 %% ===== Main Entities =====
 User {
   string id PK
   string email UK
   string password
   string name
   string institution
   string nickname UK
   string role
   string status
   timestamp deactivatedAt
   timestamp createdAt
   timestamp updatedAt
 }
 Admin
 Coach
 Contestant
 Member
 Lead {
   string user_id FK
   string group_id FK
 }
 Group {
   string id PK
   string name
 }
 Material {
   string id PK
   string group_id FK
   string url
 }
 Contest {
   string id PK
   string name
   string description
   string group_id FK
   string ownerId FK
   timestamp startTime
   timestamp endTime
   integer penalty
   timestamp createdAt
 }
 Standing {
   string id PK
   string contest_id FK
   string contestant_id FK
   integer problemsSolved
   integer totalAttempts
 }
 Register {
   string id PK
   string contestant_id FK
   string contest_id FK
 }
 Contest_Problem {
   string id PK
   string contest_id FK
   string problem_id FK
   integer order
 }
 Submission {
   string id PK
   string contestant_id FK
   string problem_id FK
   string contest_id FK
   string codeReference
   string verdict
   timestamp submittedAt
 }
 Problem {
   string id PK
   string name
   string statement
   integer timeLimit
   integer memoryLimit
   string[] tags
   string status
   string accessibility
   string authorId FK
   string testCasesFileKey
   string checkerFileKey
   string validatorFileKey
   timestamp testCasesUpdatedAt
   timestamp createdAt
   timestamp updatedAt
 }


 %% ===== Role Relationships (pseudo-entities for visualization only) =====
 User ||--o{ Admin : is
 User ||--o{ Coach : is
 User ||--o{ Contestant : is

 %% Note: Admin has implicit permissions on ALL groups
 %% without requiring registration in GroupMember or Lead (system-level permissions)

 Coach ||--o{ Lead : may_have
 Lead }o--|| Group : administrates


 Member }o--|| Group : belongs_to


 Group ||--o{ Material : has
 Group ||--o{ Contest : organizes


 Contest ||--o{ Contest_Problem : includes
 Contest_Problem }o--|| Problem : contains
 Problem }o--|| User : authored_by
 Problem ||--o{ Submission : receives
 Contest ||--o{ Register : allows
 Register }o--|| Contestant : enrolls
 Contest }o--|| User : owned_by
 Contest ||--o{ Standing : has


 Contestant ||--o{ Submission : makes
 Contestant ||--o{ Member : is
 Standing }o--|| Contestant : tracks


 Submission }o--|| Problem : solves
 Submission }o--|| Contest : may_belong_to
