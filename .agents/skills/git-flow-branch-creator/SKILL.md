---
name: git-flow-branch-creator
description: 'Intelligent Git Flow branch creator that analyzes git status/diff and creates appropriate branches following the nvie Git Flow branching model.'
---

### Instructions

```xml
<instructions>
	<title>Git Flow Branch Creator</title>
	<description>This prompt analyzes your current git changes using git status and git diff (or git diff --cached), then intelligently determines the appropriate branch type according to the Git Flow branching model and creates a semantic branch name.</description>
	<note>
		Just run this prompt and Copilot will analyze your changes and create the appropriate Git Flow branch for you.
	</note>
</instructions>
```

### Workflow

**Follow these steps:**

1. Run `git status` to review the current repository state and changed files.
2. Run `git diff` (for unstaged changes) or `git diff --cached` (for staged changes) to analyze the nature of changes.
3. Analyze the changes using the Git Flow Branch Analysis Framework below.
4. Determine the appropriate branch type based on the analysis.
5. Generate a semantic branch name following Git Flow conventions.
6. Create the branch and switch to it automatically.
7. Provide a summary of the analysis and next steps.

### Git Flow Branch Analysis Framework

```xml
<analysis-framework>
	<branch-types>
		<feature>
			<purpose>New features, enhancements, non-critical improvements</purpose>
			<branch-from>develop</branch-from>
			<merge-to>develop</merge-to>
			<naming>feature/descriptive-name or feature/ticket-number-description</naming>
			<indicators>
				<indicator>New functionality being added</indicator>
				<indicator>UI/UX improvements</indicator>
				<indicator>New API endpoints or methods</indicator>
				<indicator>Database schema additions (non-breaking)</indicator>
				<indicator>New configuration options</indicator>
				<indicator>Performance improvements (non-critical)</indicator>
			</indicators>
		</feature>

		<release>
			<purpose>Release preparation, version bumps, final testing</purpose>
			<branch-from>develop</branch-from>
			<merge-to>develop AND master</merge-to>
			<naming>release-X.Y.Z</naming>
			<indicators>
				<indicator>Version number changes</indicator>
				<indicator>Build configuration updates</indicator>
				<indicator>Documentation finalization</indicator>
				<indicator>Minor bug fixes before release</indicator>
				<indicator>Release notes updates</indicator>
				<indicator>Dependency version locks</indicator>
			</indicators>
		</release>

		<hotfix>
			<purpose>Critical production bug fixes requiring immediate deployment</purpose>
			<branch-from>master</branch-from>
			<merge-to>develop AND master</merge-to>
			<naming>hotfix-X.Y.Z or hotfix/critical-issue-description</naming>
			<indicators>
				<indicator>Security vulnerability fixes</indicator>
				<indicator>Critical production bugs</indicator>
				<indicator>Data corruption fixes</indicator>
				<indicator>Service outage resolution</indicator>
				<indicator>Emergency configuration changes</indicator>
			</indicators>
		</hotfix>
	</branch-types>
</analysis-framework>
```

### Branch Naming Conventions

```xml
<naming-conventions>
	<feature-branches>
		<format>feature/[ticket-number-]descriptive-name</format>
		<examples>
			<example>feature/user-authentication</example>
			<example>feature/PROJ-123-shopping-cart</example>
			<example>feature/api-rate-limiting</example>
		</examples>
	</feature-branches>
	<release-branches>
		<format>release-X.Y.Z</format>
		<examples>
			<example>release-1.0.0</example>
			<example>release-2.3.4</example>
		</examples>
	</release-branches>
	<hotfix-branches>
		<format>hotfix/[ticket-number-]descriptive-name or hotfix-X.Y.Z</format>
		<examples>
			<example>hotfix/security-patch</example>
			<example>hotfix-1.0.1</example>
		</examples>
	</hotfix-branches>
</naming-conventions>
```
