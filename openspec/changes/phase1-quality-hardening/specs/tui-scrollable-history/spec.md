## ADDED Requirements

### Requirement: TUI transcript SHALL be scrollable
The system SHALL render the interactive transcript through a Bubble Tea viewport so users can review conversations that exceed the visible terminal height.

#### Scenario: Transcript exceeds terminal height
- **WHEN** the conversation contains more lines than the visible TUI area
- **THEN** the TUI shows the transcript inside a scrollable viewport instead of truncating inaccessible history

#### Scenario: User scrolls through prior messages
- **WHEN** the user presses scroll navigation keys such as arrow keys, `PgUp`, `PgDn`, `Ctrl+U`, `Ctrl+D`, `Home`, or `End`
- **THEN** the viewport moves through transcript history without corrupting the current input buffer

### Requirement: Viewport dimensions SHALL react to terminal resize events
The system SHALL initialize viewport dimensions from `tea.WindowSizeMsg` and update them when the terminal size changes.

#### Scenario: First window size message initializes viewport
- **WHEN** the TUI receives its first `tea.WindowSizeMsg`
- **THEN** the viewport width and height are set to the available transcript area before rendering history

#### Scenario: Later resize updates viewport dimensions
- **WHEN** the terminal width or height changes after the TUI is already running
- **THEN** the viewport recalculates its size and continues rendering the full transcript within the new bounds

### Requirement: Streaming output SHALL preserve follow-bottom behavior
The system SHALL keep the viewport pinned to the latest output only when the user was already reviewing the bottom of the transcript before new content arrived.

#### Scenario: User is at bottom during streamed response
- **WHEN** new assistant tokens or messages are appended while the viewport is already at the bottom
- **THEN** the viewport automatically scrolls to keep the latest content visible

#### Scenario: User is reviewing older history during streamed response
- **WHEN** new assistant tokens or messages are appended while the viewport is not at the bottom
- **THEN** the viewport preserves the user's manual scroll position instead of forcing a jump to the latest content
