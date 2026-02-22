# Users Management

## Overview

Users Management handles all user-related operations in the platform, including registration, authentication support, profile management, and account lifecycle. Users are global entities that can participate in groups, contests, and submit solutions to problems.

## 🔹 User Roles

| Role | Description | Capabilities |
|------|-------------|--------------|
| `CONTESTANT` | Regular participant | Solve problems, participate in contests, join groups |
| `COACH` | Team coach/mentor | All CONTESTANT capabilities + create problems, manage groups |
| `ADMIN` | System administrator | All capabilities + user management, system configuration |

> **Note**: New users are created with `CONTESTANT` role by default. Role changes (CONTESTANT ↔ COACH) are performed by Admins. The `ADMIN` role cannot be assigned through the API.

## 🔹 User Status

| Status | Description |
|--------|-------------|
| `ACTIVE` | User can access all platform features |
| `DEACTIVATED` | User cannot authenticate or perform any actions; identity is anonymized |

> **Important**: Users are created with `ACTIVE` status. Status can only change to `DEACTIVATED` through self-deactivation or admin deactivation.

## 🔹 User Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | UUID | Auto | Unique identifier, never returned in responses |
| `email` | string | Yes | Unique email address |
| `password` | string | Yes | Hashed password, never returned in responses |
| `name` | string | Yes | User's full name |
| `nickname` | string | Yes | Display name/alias, stored in lowercase, unique |
| `country` | string | Yes | User's country |
| `city` | string | Yes | User's city |
| `institution` | string | Yes | User's institution or organization |
| `role` | enum | Auto | ADMIN, COACH, or CONTESTANT (default: CONTESTANT) |
| `status` | enum | Auto | ACTIVE or DEACTIVATED (default: ACTIVE) |
| `createdAt` | timestamp | Auto | Account creation timestamp |
| `updatedAt` | timestamp | Auto | Last modification timestamp (nullable) |
| `deactivatedAt` | timestamp | Auto | Deactivation timestamp (nullable) |

## 🔹 Password Requirements

All passwords must meet these security requirements:

- Minimum **8 characters**
- At least **1 uppercase letter** (A-Z)
- At least **1 special character** (!@#$%^&*()_+-=[]{}|;:',.<>?/)
- At least **1 number** (0-9)

## 🔹 Nickname Handling

- Nicknames are stored in **lowercase** regardless of input case
- Nickname lookups are **case-insensitive**
- Nicknames must be **unique** across all active users
- When a user is deactivated, nickname is anonymized to `user_anonimo_{10-char-uuid}`
- Original nickname becomes available for new registrations after deactivation

## 🔹 Global Group Membership

Upon registration, every user is automatically added as a member of the **Global Group**. This ensures all users have access to platform-wide contests and resources.

## 🔹 Verification Codes

Several features use 6-digit numeric verification codes sent via email:

| Feature | Code Expiration | Rate Limit |
|---------|-----------------|------------|
| Password Recovery | 15 minutes | 5 requests per email per hour |
| Email Change | 15 minutes | 1 active code per user |
| Account Deactivation | 15 minutes | 5 confirmation attempts, then 1-hour block |

## 🔹 Account Deactivation

### Self-Deactivation
- Only **COACH** and **CONTESTANT** can self-deactivate
- **ADMIN** cannot deactivate their own account
- Requires confirmation code sent to email

### Admin Deactivation
- Admins can deactivate any non-admin user via `POST /admin/users/{id}/deactivate`
- Used for banning users or handling compromised accounts
- Does not require confirmation code (unlike self-deactivation)

### Deactivation Effects
1. Status changes to `DEACTIVATED`
2. Email is **unlinked** (set to NULL) and becomes available for new registrations
3. Nickname is **anonymized** to `user_anonimo_{uuid}`
4. All sessions are invalidated
5. User **cannot authenticate** through any method
6. Historical content (submissions, standings) is preserved but displays anonymized identity

## Implemented Specs

| Spec | Endpoint | Description |
|------|----------|-------------|
| [Create user](Create%20user/spec.md) | `POST /users` | Self-registration (CONTESTANT role only) |
| [Get user information](Get%20user%20information/spec.md) | `GET /users/me`, `GET /users/{nickname}` | View own or others' profiles |
| [Update data user](Update%20data%20user/spec.md) | `PUT /users` | Update own profile (name, nickname, institution) |
| [Update email user](Update%20email%20user/spec.md) | `POST /users/email-change/*` | Change email with verification |
| [Update password](Update%20password/spec.md) | `PUT /users/password` | Change own password |
| [Recover password](Recover%20password/spec.md) | `POST /password/*` | Password recovery flow |
| [Admin update user](Admin%20update%20user/spec.md) | `PUT /admin/users/{id}` | Admin updates user data/role |
| [Self deactivate user](Self%20deactivated%20user/spec.md) | `POST /users/deactivation/*` | Self-deactivation with confirmation |
| [Admin deactivate user](Admin%20deactivate%20user/spec.md) | `POST /admin/users/{id}/deactivate` | Admin deactivates any user |
| [List users](List%20users/spec.md) | `GET /admin/users` | Admin lists all users with filters and search |
| [User activity dashboard](User%20activity%20dashboard/spec.md) | `GET /users/me/dashboard` | View personal activity dashboard with statistics |

## Future Specs (Planned)

| Spec | Endpoint | Description |
|------|----------|-------------|
| Admin reactivate user | `POST /admin/users/{id}/reactivate` | Admin reactivates deactivated user |

## Permission Matrix

| Action | CONTESTANT | COACH | ADMIN |
|--------|------------|-------|-------|
| Register (create account) | ✅ | ✅ | ✅ |
| View own profile | ✅ | ✅ | ✅ |
| View own dashboard | ✅ | ✅ | ✅ |
| View other active profiles | ✅ (public fields) | ✅ (public fields) | ✅ (all fields) |
| View Admin profiles | ❌ | ❌ | ✅ |
| Update own profile | ✅ | ✅ | ✅ |
| Update own email | ✅ | ✅ | ✅ |
| Update own password | ✅ | ✅ | ✅ |
| Self-deactivate | ✅ | ✅ | ❌ |
| Update other users | ❌ | ❌ | ✅ |
| Deactivate other users | ❌ | ❌ | ✅ |
| List all users | ❌ | ❌ | ✅ |

## Profile Visibility Rules

| Requester | Own Profile | Other Active Users | Admin Profiles | Deactivated Users |
|-----------|-------------|-------------------|----------------|-------------------|
| CONTESTANT/COACH | All fields (except password, id) | Public fields only | 403 Forbidden | 404 Not Found |
| ADMIN | All fields (except password, id) | All fields (except password, id) | All fields (except password, id) | 404 Not Found |

**Public fields**: name, nickname, institution, role, createdAt

**Sensitive fields**: email (only visible to self or Admin)

## Security Features

### Session Invalidation
Sessions are invalidated when:
- Password is changed (all sessions including current)
- Password is recovered/reset (all sessions)
- Account is deactivated (all sessions)

### Rate Limiting
- Password recovery: 5 requests per email per hour
- Password update: 5 failed attempts, then 1-hour cooldown
- Deactivation confirmation: 5 attempts per code, then 1-hour block

### Email Notifications
Notifications are sent for:
- Successful email change (to both old and new email)
- Successful password change
- Account deactivation confirmation

---

## Implementation Dependencies

```
Users Management (base)
    ↓
Create User ✅
    ↓
Get User Information ✅
    ↓
Update Data User ✅
    ↓
Update Email User ✅
    ↓
Update Password ✅
    ↓
Recover Password ✅
    ↓
Admin Update User ✅
    ↓
Self Deactivate User ✅
    ↓
Admin Deactivate User ✅
```

## Key Design Decisions

### Why separate endpoints for email/password changes?
- **Security**: Each sensitive operation requires its own verification flow
- **Audit trail**: Separate endpoints allow better logging and monitoring
- **User experience**: Clear, focused flows for each action

### Why Admin cannot self-deactivate?
- **Safety**: Prevents accidental lockout of administrative access
- **Governance**: Admin accounts require oversight from other admins

### Why unlink email on deactivation?
- **Privacy**: User's email is no longer associated with platform data
- **Reusability**: Email can be used to create a new account if user returns
- **Compliance**: Supports "right to be forgotten" requirements

### Why anonymize nickname instead of delete?
- **Data integrity**: Historical records (standings, submissions) remain valid
- **Fairness**: Contest results and rankings are preserved
- **Audit**: Platform can still track activity patterns without exposing PII


