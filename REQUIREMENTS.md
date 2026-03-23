# Coach Assist - Project Requirements & Vision

## Overview
**Coach Assist** is an API-driven application designed to streamline administrative tasks associated with coaching bodyflight at wind tunnel workshops (both solo and 4-way skills). By automating organizational tasks, coaches can focus their energy on core responsibilities: evaluating student experience levels and creating flight plans that challenge them appropriately.

To ensure the tool is accessible to all coaches without creating "tech support hell" (e.g., forcing them to install CLI binaries or configure workspace tools), the project will be built in distinct phases:
* **Phase I:** A Terminal User Interface (TUI) tailored for the primary developer.
* **Phase II:** A Web Application for the broader coaching team.

## Architecture & Principles
* **Language:** Go (Backend/API and Phase I TUI).
* **API-First Design:** Core business logic (ingestion, Google Workspace automation, email drafting) will be exposed via a clean, internal API boundary. Both the Phase I TUI and Phase II Web App will consume these unified core services.
* **Testing:** High priority on writing robust tests early and often to ensure long-term maintainability and prevent regressions.

## External Interfaces & Dependencies
* **Google Workspace CLI (`gws`):** Used to interact with Google Drive and manage shared folders/sheets. The architecture should isolate this dependency so that if `gws` proves inadequate, we can pivot to using the Google Drive/Docs APIs directly without rewriting the core application logic.
  * *Constraint Mitigation (Environment Security):* On macOS, `gws` can experience strict keychain or SSL certificate sandboxing issues. The app execution wrapper should export `GOOGLE_WORKSPACE_CLI_KEYRING_BACKEND=file` and explicitly map `SSL_CERT_FILE` to reliably bypass local OS interference.
* **Local Input Files:** Ability to read and parse Excel files (`.xlsx` or `.csv` if exported) created by the workshop organizer.

## Core Features & Workflows

### 1. Data Ingestion & Parsing
The application must extract relevant coach-to-student assignments and timing from the organizer's input files:
* **Gmail Attachment Extraction:** The system will locate target emails using `has:attachment`. When fetching the literal binary payloads via the Gmail API, the system must properly decode the payload from Google's Base64Url encoding format before writing standard `.xlsx` files to the local disk.
* **Interactive Disambiguation:** The system handles data ingestion by scanning the coach's Gmail for specific organizer subjects. Because replies in an email thread can lack the original attachment, queries strictly use `has:attachment`. If a search yields multiple attachment-bearing instances (like multiple revisions of the schedule), the TUI must format the snippets/dates and interactively prompt the user to select the authoritative source document.
* **Schedule File:** An Excel file detailing the workshop schedule. Because the schedule often changes leading up to the event due to cancellations or modifications, the tool must reliably parse updated versions to pinpoint exactly who the user is coaching and when.
* **Roster File:** An Excel workbook containing two distinct sheets:
  * **Email List:** Contact information (email addresses) for all participants.
  * **Coach Assignments ("Who is making dives"):** Mapping of participants to their assigned plan-maker/coach.
  * *Constraint Mitigation (Fuzzy String Matching):* The schedule usually contains shorthand names (e.g., "Kyle H / Joe N 4way") while the roster contains formal names ("Kyle Hermberg"). The extraction pipeline must utilize advanced substring heuristics (checking dual-initials, text prefixes, and partial matches) to accurately cross-reference these records. Because unstructured text parsing is never 100% foolproof, the TUI must prominently display the generated matches so the coach can spot-check them before broadcasting emails. If an email cannot be confidently found, the system should output explicit empty brackets `[]` to visually alert the user to the missing field.

### 2. Google Workspace Automation
Coaches use a shared Google Drive to ensure all flight plans are available in a well-known format (as coaches sometimes cover for each other). For each assigned student/group, the app should leverage `gws` to:
* **Get or Create Folders:** Create a new folder on the shared Google Drive named after the assigned student(s) or team.
  * *Constraint Mitigation (Duplicate Names & Strict Scoping):* Because Google Drive does not enforce unique paths, the API will blindly create duplicate folders. To prevent this, the core logic must first `list` the directory to see if the folder already exists. Crucially, this query MUST be explicitly scoped using exactly `'[PARENT_ID]' in parents` to avoid accidentally finding an identically named folder from a past workshop elsewhere in the Drive. For 1-on-1s, this parent is the `workshop_parent_folder_id`. For 4-way groups, this parent MUST be strictly mapped to the `teams_folder_id` (e.g. the 'AAA - Teams' directory). If the folder already exists within that specific parent, the software must gracefully recycle the existing folder's ID instead of generating a confusing duplicate.
* **Copy Templates:** Copy a standardized template sheet into the student/team's new folder and rename it appropriately.
* **Dynamic Spreadsheet Modification:** Populate the newly copied Google Sheet with variables like Student Name, Workshop Date, and Coach automatically.
  * *Constraint Mitigation (Layout Shifts):* Hardcoding cell coordinates (e.g., `B2`) makes the application too fragile if the template layout changes. The API must query the spreadsheet's 2D grid to search for specific text headers (e.g., "Name:") to anchor input locations.
  * *Constraint Mitigation (Merged Cells - Silent Failures):* Google Sheets often use merged cells for headers (e.g., horizontally combining `A` and `B`). The API must query the `merges` array metadata to identify the exact width of a header block. To successfully write values, the tool must calculate the `endColumnIndex` of the block to skip entirely past the merge, rather than blindly adding `+1` column, which causes silent write failures directly into the hidden merged span.
* **Manage Permissions:** Update the sharing permissions on the newly created flight plan sheet so it can be viewed by "anyone with the link".

### 3. Communication Workflows
The application should draft standardized emails to the participants:
* **Strict Constraint (No Automated Sending):** The application MUST NEVER attempt to send emails automatically through the Gmail API or any other SMTP server. The coach uses a separate business domain for communication. The TUI must exclusively render the "To", "Subject", and "Body" fields as plain, easily copy-pasteable text (or local `.txt` files / `mailto:` links) so the coach can manually send them from their correct business client.
* **Configurable Templates & TUI Selection:** Standardized verbiage should be fully extracted into arrays inside `config.json`. The application must handle cycling multiple templates (e.g. "Standard" vs "New Student") and allow the user to easily toggle between these template variations within the TUI before copying the draft.
* **Initial Outreach / Discovery:** Draft an email to ask the student about their goals, what they hope to work on, and their current experience level.
  * *Contextual Rule:* If it is a 4-way group, the email should specifically ask which "slot" the person would like to fly.
* **Plan Delivery & Logistics:** Draft a pre-workshop email containing brief introductory words, the viewable link to the generated Google Sheet plan, and a reminder of the date and time they are expected to arrive and meet at the tunnel.

### 4. User Interfaces (Phased Rollout)
* **API Foundation:** The core application will operate as a unified, stateless backend service/library, strictly decoupling internal orchestration from presentation. The API boundary MUST be inherently resilient enough to flawlessly support three distinct consumers:
* **Phase I - Terminal User Interface (TUI):** An interactive menu-driven TUI tailored for the primary developer. Key TUI capabilities include:
  * **Workflow Guidance:** Fast, keyboard-centric menus that walk the user through parsing the schedule, building folders, and drafting emails.
  * **Dynamic Template Toggling:** The TUI must provide a fast hotkey (e.g., `[TAB]`) that allows the user to instantly cycle the view between the different templates stored in `config.json` (like Standard vs New Student) before copying the draft.
  * **Profile Management:** The interface will persistently display the currently active "coach profile" (whoami) on the screen. It must also provide a dedicated menu option to update or swap this identity configuration smoothly on the fly.
* **Phase I - Scripting CLI:** A dedicated headless Command Line Interface structured exclusively for fast, single-shot automation sequences and robust CI/scripting integrations.
* **Phase II - Web Application:** A browser-based GUI natively wrapping the same core API logic, allowing other coaches to securely execute the pipelines without installing local binaries or generating CLI tokens.

### 5. Notes & Planning
* **Planning Call Notes:** A lightweight way to record notes from planning calls regarding people and groups. (As an MVP, this can simply be a dedicated text file or directory in the workspace where the coach can jot down unstructured notes, though it could later be integrated into the TUI).

### 6. Configuration
To ensure the tool remains flexible across different workshops, organizations, or seasons, all external IDs will be extracted into a central configuration file (e.g., `config.json`). The application will read this on startup to determine:
* **Workshop Scope:** Volatile settings that change per weekend, such as `cc_emails` for active coaches. The TUI must explicitly remind the user to verify this CC list before launching the email generation workflows.
* **Active Coach Profile (`active_coach`):** A string pointing to the identity of the currently operating coach within the `coaches` dictionary.
* **Coach Dictionary (`coaches`):** A collection mapping every coach (e.g., `Greg`, `Doug`) to their highly personalized configuration block. This map specifically isolates each coach's formal `signature`, target `email`, and highly-specific `email_templates` dictionary (ensuring one coach's preferred outreach verbiage does not pollute another's).
* **Parent Directories:** The Google Drive Folder ID of the "Shared Workshop" folder where new student folders should be created.
* **Template Documents:** A mapping of template names to their Google Drive file IDs (e.g., the *Individual Skills Worksheet*).

### 7. Core State Persistence (Resuming Workflow)
To ensure the coach can safely exit and reopen the TUI at any time throughout the weekend, the overarching system must aggressively persist state:
* **File Presence Checks:** Upon boot, the Application must poll the local filesystem for imported Gmail artifacts. Whenever attachments are cleanly downloaded from Gmail, the ingestion engine will explicitly copy them to standardized `latest-schedule.xlsx` and `latest-roster.xlsx` filenames. If these canonical files are physically present on launch, their menu components are automatically loaded and marked complete (`[✅]`).
* **Flight Plan State Cache:** A lightweight proxy file (e.g., `state.json` or local memory-map) must be continuously maintained alongside the configuration. This persists the core `FlightPlan` arrays so that structural tracking booleans (like `IsWorkspaceCreated`, `IsDiscoveryDrafted`, and dynamically acquired Google Drive IDs) natively survive TUI crashes or manual exits.

### 8. Architectural Constraints & TUI Learnings
Throughout the Phase I Go implementation, several critical technical decisions were structurally enforced to bypass local Operating System bounds and ensure UX fluidity:
* **macOS Sandbox Isolation Bypasses:** When attempting to compile Go or execute automated credentials in the background CLI context, aggressive macOS sandbox policies physically blocked access to root temporary caches and the global GUI keychain (triggering pure `Operation not permitted (os error 1)` blocks). The `coach-assist` application completely bypasses these errors natively by mapping internal caches to the strict local project runtime (`GOCACHE=$PWD/.go-cache`) and aggressively passing the `GOOGLE_WORKSPACE_CLI_KEYRING_BACKEND=file` flag natively downstream on all `gws` subprocess invocations.
* **Asynchronous `tview` Event Spooling:** Because the Google Workspace JSON ingestion matrix requires multiple nested physical payload requests scaling perfectly across massive 3MB+ Excel sheets, synchronous execution fully locked the UI thread. The application officially enforces decoupled GoRoutines injecting trace bytes securely back into native `tview.TextView` logging modals via `app.QueueUpdateDraw()`. This completely prevents application freezing and provides the user with beautiful real-time validation endpoints.
* **Caching File Provenance (Ingestion State):** Because the backend natively sanitizes all downloaded attachments to `latest-schedule.xlsx` and `latest-roster.xlsx`, the TUI instantly loses the context of *exactly which* email threads those files originated from. The architecture strictly solves this by formally streaming the active `"Gmail Subject Query"` into `state.json` upon successful payload extraction. This guarantees the interactive Dashboard visually reconstructs the EXACT context of the local `latest-*` files flawlessly across reboots!
