# Tasks: Improve Tests

**Input**: Design documents from `/specs/001-improve-tests/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are not explicitly requested in the feature specification, but test infrastructure improvements inherently involve testing the test infrastructure itself.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: Repository root structure
- Test infrastructure improvements affect: `test/`, `.github/workflows/`, `Makefile`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure for test infrastructure

- [X] T001 Create test directory structure: `test/unit/` and `test/integration/` directories
- [X] T002 [P] Create `.github/workflows/` directory if it doesn't exist
- [X] T003 [P] Verify Go 1.25.1 is available and matches go.mod

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Create Makefile target `test-coverage` in Makefile for local coverage analysis
- [X] T005 [P] Create Makefile target `test-bench` in Makefile for running benchmarks
- [X] T006 [P] Create coverage parsing script or function to extract overall and per-package percentages from `go tool cover -func` output
- [X] T007 Verify existing test infrastructure (Ginkgo/Gomega setup) is working

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Achieve Minimum Test Coverage (Priority: P1) 🎯 MVP

**Goal**: Implement 80% test coverage enforcement with CI integration and local command

**Independent Test**: Run `make test-coverage` locally and verify coverage percentage is reported. Open a PR and verify CI runs coverage check and blocks merge if coverage < 80%.

### Implementation for User Story 1

- [X] T008 [US1] Implement `test-coverage` Makefile target in Makefile that runs `go test -coverprofile=coverage.out ./...`
- [X] T009 [US1] Add coverage report generation to `test-coverage` target in Makefile using `go tool cover -func=coverage.out`
- [X] T010 [US1] Add coverage percentage parsing and display to `test-coverage` target in Makefile (overall percentage and per-package breakdown)
- [X] T011 [US1] Add HTML coverage report generation to `test-coverage` target in Makefile using `go tool cover -html=coverage.out -o coverage.html`
- [X] T012 [US1] Create GitHub Actions workflow file `.github/workflows/test-coverage.yml` with basic structure
- [X] T013 [US1] Add Go setup step to `.github/workflows/test-coverage.yml` workflow
- [X] T014 [US1] Add test coverage analysis step to `.github/workflows/test-coverage.yml` that runs `go test -coverprofile=coverage.out ./...`
- [X] T015 [US1] Add coverage report generation step to `.github/workflows/test-coverage.yml` using `go tool cover -func=coverage.out`
- [X] T016 [US1] Add coverage threshold check to `.github/workflows/test-coverage.yml` that fails if overall coverage < 80%
- [X] T017 [US1] Add coverage reporting to workflow summary in `.github/workflows/test-coverage.yml` (overall percentage and per-package breakdown)
- [X] T018 [US1] Configure workflow triggers in `.github/workflows/test-coverage.yml` for PR events (opened, synchronize, reopened)
- [X] T019 [US1] Add clear error message to `.github/workflows/test-coverage.yml` when coverage threshold is not met

**Checkpoint**: At this point, User Story 1 should be fully functional - coverage can be checked locally and CI enforces 80% threshold

---

## Phase 4: User Story 2 - Organize Tests into Standard Structure (Priority: P2)

**Goal**: Reorganize all existing tests into `test/unit/` and `test/integration/` structure via big-bang migration

**Independent Test**: Verify all unit tests are in `test/unit/` and all integration tests are in `test/integration/`. Run `go test ./test/unit/...` and `go test ./test/integration/...` to confirm tests still pass.

### Implementation for User Story 2

- [ ] T020 [US2] Identify all existing test files in the codebase (find all `*_test.go` files)
- [ ] T021 [US2] Classify each test file as unit or integration based on dependencies (mocks vs external services)
- [ ] T022 [US2] Create package directory structure in `test/unit/` mirroring source structure (e.g., `test/unit/internal/controller/k8s/`)
- [ ] T023 [US2] Create package directory structure in `test/integration/` organized by feature (e.g., `test/integration/k8s/`)
- [ ] T024 [US2] Move unit test files from source directories to `test/unit/` maintaining package structure
- [ ] T025 [US2] Move integration test files to `test/integration/` organized by feature
- [ ] T026 [US2] Update import paths in moved test files to reflect new locations
- [ ] T027 [US2] Update package declarations in moved test files to match new locations
- [ ] T028 [US2] Verify all moved tests compile correctly with updated imports
- [ ] T029 [US2] Run all moved tests to verify they still pass: `go test ./test/unit/...` and `go test ./test/integration/...`
- [ ] T030 [US2] Update Makefile `test` target if needed to reflect new test locations
- [ ] T031 [US2] Remove old test files from source directories after successful migration

**Checkpoint**: At this point, User Story 2 should be complete - all tests are organized in the new structure and passing

---

## Phase 5: User Story 3 - Ensure Business Logic Has Comprehensive Test Coverage (Priority: P2)

**Goal**: Achieve 100% test coverage for all exported functions in `internal/controller/*/service/` layer (all service layers within controllers)

**Independent Test**: Run coverage analysis and verify all `internal/controller/*/service/` layers show 100% coverage. Check that all exported functions have behavioral test coverage.

### Implementation for User Story 3

**Implementation Note**: Go's `go test` command uses `...` to match package patterns recursively. The pattern `./internal/controller/.../service/...` will match all service layers under any controller (e.g., `internal/controller/flow/service/`, `internal/controller/k8s/service/`, etc.). This supports the wildcard pattern `internal/controller/*/service/` specified in the constitution and spec.

- [ ] T032 [US3] Run coverage analysis to identify gaps in `internal/controller/*/service/` layer: `go test -coverprofile=coverage.out ./internal/controller/.../service/...` (Go's `...` pattern matches all service layers under any controller)
- [ ] T033 [US3] Generate HTML coverage report for `internal/controller/*/service/` layer: `go tool cover -html=coverage.out -o coverage-service.html`
- [ ] T034 [US3] Identify all uncovered exported functions in `internal/controller/*/service/` layer from coverage report (filter coverage report to show only service layer packages)
- [ ] T035 [US3] Create test files in `test/unit/internal/controller/*/service/` for uncovered functions (mirror source structure: e.g., `test/unit/internal/controller/flow/service/` for `internal/controller/flow/service/`)
- [ ] T036 [US3] Write tests for uncovered exported functions in `test/unit/internal/controller/*/service/` using Ginkgo/Gomega (mirror source structure for each service layer)
- [ ] T037 [US3] Ensure tests cover both success and error paths for each function
- [ ] T038 [US3] Ensure tests exercise all branches in functions with multiple branches
- [ ] T039 [US3] Verify all tests pass: `go test ./test/unit/internal/controller/.../service/...` (tests all service layer tests under any controller)
- [ ] T040 [US3] Re-run coverage analysis to confirm 100% coverage for all `internal/controller/*/service/` layers
- [ ] T041 [US3] Add service layer coverage check to `test-coverage` Makefile target in Makefile (report but don't block)
- [ ] T042 [US3] Add service layer coverage reporting to `.github/workflows/test-coverage.yml` workflow (warning if < 100%)

**Checkpoint**: At this point, User Story 3 should be complete - all `internal/controller/*/service/` layers have 100% coverage

---

## Phase 6: User Story 4 - Add Table-Driven Tests for Complex Functions (Priority: P3)

**Goal**: Convert functions with 3+ input variants to use table-driven test patterns

**Independent Test**: Identify functions with multiple input variants and verify they use table-driven test patterns. Verify new test cases can be added as table rows.

### Implementation for User Story 4

- [ ] T043 [US4] Identify all functions with 3+ input variants across the codebase via manual code review of existing test files (look for functions with multiple test cases or repeated test patterns)
- [ ] T044 [US4] Review existing tests for identified functions to determine which need table-driven conversion
- [ ] T045 [US4] Convert first function's tests to table-driven pattern in appropriate test file in `test/unit/` or `test/integration/`
- [ ] T046 [US4] Convert second function's tests to table-driven pattern using Ginkgo's `DescribeTable` in appropriate test file
- [ ] T047 [US4] Convert remaining functions' tests to table-driven patterns in their respective test files
- [ ] T048 [US4] Verify all table-driven tests execute and report individual pass/fail status for each table row
- [ ] T049 [US4] Verify all table-driven tests pass: run full test suite
- [ ] T050 [US4] Document table-driven test pattern in test files with comments for future reference

**Checkpoint**: At this point, User Story 4 should be complete - functions with multiple variants use table-driven tests

---

## Phase 7: User Story 5 - Add Performance Benchmarks (Priority: P3)

**Goal**: Add performance benchmarks for critical functions (Kubernetes API calls, GCP API calls, heavy computation)

**Independent Test**: Run `make test-bench` and verify benchmarks execute and report performance metrics. Verify benchmarks run in CI (advisory only, don't block).

### Implementation for User Story 5

- [ ] T051 [US5] Identify critical functions requiring benchmarks: Kubernetes API calls, GCP API calls, heavy computation
- [ ] T052 [US5] Create benchmark file for Kubernetes API client operations: `test/unit/pkg/k8s/utils/client/k8s_bench_test.go` or alongside source
- [ ] T053 [US5] Implement `BenchmarkK8sClient_UpdateDeployment` benchmark function in benchmark file
- [ ] T054 [US5] Create benchmark file for GCP API client operations: `test/unit/pkg/gcp/utils/client/gcp_bench_test.go` or alongside source
- [ ] T055 [US5] Implement `BenchmarkGCPClient_ScaleVMInstance` benchmark function in benchmark file
- [ ] T056 [US5] Create benchmark file for period calculations: `test/unit/pkg/period/period_bench_test.go` or alongside source
- [ ] T057 [US5] Implement `BenchmarkPeriodCalculator_CalculateNextPeriod` benchmark function in benchmark file
- [ ] T058 [US5] Add additional benchmarks for other critical functions identified in T051
- [ ] T059 [US5] Verify all benchmarks run successfully: `go test -bench=. -benchmem ./...`
- [ ] T060 [US5] Add benchmark step to `.github/workflows/test-coverage.yml` workflow (advisory, doesn't block)
- [ ] T061 [US5] Add benchmark reporting to workflow summary in `.github/workflows/test-coverage.yml` (performance metrics)

**Checkpoint**: At this point, User Story 5 should be complete - benchmarks exist for critical functions and run in CI

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

**Note**: Mock verification (FR-009) is implicit in test writing tasks T036-T038; existing tests already use interface-based mocking as verified in T007.

- [ ] T062 [P] Update README.md with test infrastructure documentation (coverage commands, test organization)
- [ ] T063 [P] Verify all Makefile targets work correctly: `make test-coverage`, `make test-bench`, `make test`
- [ ] T064 [P] Verify CI workflow is set as required status check in repository settings
- [ ] T065 [P] Test full workflow: create test PR and verify coverage enforcement works
- [ ] T066 [P] Update quickstart.md validation: verify all commands in quickstart.md work correctly
- [ ] T067 [P] Code cleanup: remove any temporary files or scripts used during migration
- [ ] T068 [P] Document test organization structure in README.md or test/README.md
- [ ] T069 [P] Measure and verify coverage analysis completes within 30 seconds (SC-007): run `time make test-coverage`
- [ ] T070 [P] Measure and verify unit test suite executes in under 5 seconds (SC-008): run `time go test ./test/unit/...`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Independent, but benefits from US1 coverage tooling
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Depends on US2 (tests must be in new structure)
- **User Story 4 (P3)**: Can start after Foundational (Phase 2) - Depends on US2 (tests must be in new structure)
- **User Story 5 (P3)**: Can start after Foundational (Phase 2) - Independent, can run in parallel with others

### Within Each User Story

- Setup tasks before implementation tasks
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel (T002, T003)
- All Foundational tasks marked [P] can run in parallel (T005, T006)
- User Stories 1, 2, and 5 can start in parallel after Foundational (US3 and US4 depend on US2)
- Within US2: Directory creation tasks (T022, T023) can run in parallel
- Within US4: Multiple function conversions (T045-T047) can run in parallel
- Within US5: Multiple benchmark file creations (T052, T054, T056) can run in parallel
- All Polish tasks marked [P] can run in parallel

---

## Parallel Example: User Story 1

```bash
# Setup tasks that can run in parallel:
Task: "Add Go setup step to .github/workflows/test-coverage.yml workflow"
Task: "Add test coverage analysis step to .github/workflows/test-coverage.yml"

# Coverage reporting tasks that can run in parallel:
Task: "Add coverage report generation step to .github/workflows/test-coverage.yml"
Task: "Add coverage reporting to workflow summary in .github/workflows/test-coverage.yml"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Coverage enforcement)
4. **STOP and VALIDATE**: Test coverage enforcement works locally and in CI
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP - coverage enforcement!)
3. Add User Story 2 → Test independently → Deploy/Demo (test organization)
4. Add User Story 3 → Test independently → Deploy/Demo (service layer coverage)
5. Add User Story 4 → Test independently → Deploy/Demo (table-driven tests)
6. Add User Story 5 → Test independently → Deploy/Demo (benchmarks)
7. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (coverage enforcement)
   - Developer B: User Story 2 (test reorganization) → then User Story 3 (service coverage)
   - Developer C: User Story 5 (benchmarks) → then User Story 4 (table-driven tests)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
- Test infrastructure improvements don't require traditional "models" or "services" - focus on tooling and organization
