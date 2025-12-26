erDiagram


 %% ===== Entidades principales =====
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
 Admin_Group {
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
   string group_id FK
   string ownerId FK
   timestamp startTime
   timestamp endTime
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
   string authorId FK
   string testCasesFileKey
   string checkerFileKey
   string validatorFileKey
   timestamp testCasesUpdatedAt
   timestamp createdAt
   timestamp updatedAt
 }


 %% ===== Relaciones de roles (pseudo-entidades sólo para visualización) =====
 User ||--o{ Admin : is
 User ||--o{ Coach : is
 User ||--o{ Contestant : is


 Admin ||--o{ Admin_Group : may_have
 Coach ||--o{ Admin_Group : may_have
 Admin_Group }o--|| Group : administrates


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
