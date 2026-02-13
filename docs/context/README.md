# Egress Service System Context

## Overview

The Egress service is a critical component within a Trusted Research Environment (TRE) that controls the flow of files and data out of the secure environment. It ensures that sensitive data is not copied out without proper approval from authorised Egress checkers.

## Dynamic Context

The following sequence diagram illustrates the key external interactions with the Egress service.

```mermaid
sequenceDiagram
    actor researcher as Researcher
    actor checker as Egress Checker
    participant tre as TRE
    box rgb(248,252,228) Egress App
        participant frontend as Egress App Frontend
        participant backend as Egress App Backend
    end
    participant egress as Egress Service
    participant db as Database
    participant S3 as Storage

    Note over researcher,S3: Flow: Request Egress Approval

    researcher->>tre: Connect to TRE
    tre->>S3: Move files for egress approval
    researcher->>checker: Request approval

    Note over researcher,S3: Flow: Request Egress File List

    checker->>frontend: Login (Checker)
    frontend->>backend: Authenticate user
    backend-->>frontend: User logged in

    checker->>frontend: Request egress files
    frontend->>backend: Get egress files
    backend->>egress: list-files<br/>{HTTP-creds, project-id, files-location}
    egress->>S3: Get files
    S3-->>egress: File list
    egress->>db: Get file metadata
    db-->>egress: Metadata
    egress-->>backend: Files with metadata
    backend-->>frontend: Files with metadata
    frontend-->>checker: Files awaiting approval

    Note over researcher,S3: Flow: Approve File for Egress

    checker->>tre: Connect to TRE
    tre->>S3: Request file for viewing
    S3-->>tre: File content
    checker->>tre: View file
    checker->>frontend: Approve file
    frontend->>backend: Approve file<br/>{file-id}
    backend->>egress: approve-file<br/>{HTTP-creds, project-id, file-id, user-id}
    egress->>db: Record approval
    db-->>egress: Approval recorded OK
    egress-->>backend: Success
    backend-->>frontend: Approval confirmed

    Note over researcher,S3: Flow: Download File

    researcher->>frontend: Login (Researcher)
    frontend->>backend: Authenticate user
    backend-->>frontend: User logged in

    researcher->>frontend: Download file
    frontend->>backend: Download file<br/>{file-id}
    backend->>egress: Download file<br/>{HTTP-creds, project-id, file-id,<br/>files-location, required-approvals, max-file-size}
    egress->>db: Get approval status
    db-->>egress: Approval status
    egress->>S3: Get file
    S3-->>egress: File stream
    egress-->>backend: Stream file
    backend-->>frontend: Stream file
```

### Actors and Participants

- **Researcher**: User working with sensitive data in the TRE. Researcher typically initiates the process to egress one or more files in the TRE.
- **Egress Checker**: User authorised to approve files for egress.
- **TRE**: Truest Research Environment where Researcher works with sensitve data. Egress Checker also has access to the TRE to allow them to review content of files prior to approving them for egress.
- **Egress App Frontend**: Frontend portion of the web application used by Egress Checker to approve egress requests. This is likely a singe-page web frontend that communicates with the backend of the web app.
- **Egress App Backend**: Backend of the web application used by Egress Checker. The backend manages configuration (HTTP credentials, file location, approval thresholds, etc.) and communicates with the Egress service API. The backend also handles user authentication.
- **Egress Service**: This service that provides core Egress functionality.

### Dependencies

- **Database**: Stores egress request metadata and approval status.
- **Storage**: Stores files pending egress as well those that have been approved for egress.

## Static Contextual View

The following diagram provides a static view of the Egress service and its relationships with external systems and actors.

```mermaid
C4Context
    title Egress Service Context

    Person(researcher, "Researcher", "Works with sensitive data in TRE")
    Person(checker, "Egress Checker", "Approves files for egress from TRE")

    System_Boundary(tre_boundary, "Trusted Environment") {
        System(tre, "TRE", "Environment for working with sensitive data")
        System(egress, "Egress Service", "Egress functionality for TRE")
        SystemDb(db, "Database", "Stores egress metadata and approvals")
        SystemDb(storage, "Storage", "Storage for egress files")
    }

    System_Boundary(egress_app, "Egress App") {
        Component(frontend, "Frontend", "Web UI", "Web app for approving egress requests")
        Component(backend, "Backend", "Web Backend", "Manages authentication, configuration<br/>and communicates with Egress Service")
    }

    Rel(researcher, tre, "Works with sesitive data")
    Rel(researcher, checker, "Requests egress approval")
    Rel(researcher, frontend, "Downloads approved files")
    Rel(checker, tre, "Reviews egress files")
    Rel(checker, frontend, "Approves egress files")

    Rel(frontend, backend, "Delegates user actions")
    Rel(backend, egress, "Invokes Egress service API")

    Rel(tre, storage, "Place egress files<br/>Read egress files")
    Rel(egress, db, "Read/write egress metadata")
    Rel(egress, storage, "Read/write egress files")
```
