# Tasks: Rewrite GCP Controller Using Chain of Responsibility Pattern

**Input**: Design documents from `/specs/001-gcp-controller-chain/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are explicitly required by the constitution (TDD for service layer) and feature specification (SC-003: 80% test coverage, SC-004: handler tests <100ms).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: Repository root structure
- Refactoring affects: `internal/controller/gcp/` and new `internal/controller/gcp/service/` directory

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure for handler chain infrastructure

- [X] T001 Create service directory structure: `internal/controller/gcp/service/` and `internal/controller/gcp/service/handlers/` directories
- [X] T002 [P] Verify Go 1.25.1 is available and matches go.mod
- [X] T003 [P] Verify existing dependencies are available: controller-runtime, zerolog, ginkgo/gomega

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Create Handler interface definition in `internal/controller/gcp/service/interfaces.go`
- [X] T005 [P] Create ReconciliationContext structure in `internal/controller/gcp/service/context.go`
- [X] T006 [P] Create ReconciliationResult structure in `internal/controller/gcp/service/context.go`
- [X] T007 [P] Create error categorization types (CriticalError, RecoverableError) in `internal/controller/gcp/service/errors.go`
- [X] T008 Create Chain interface and implementation in `internal/controller/gcp/service/chain.go`
- [X] T009 Implement chain execution logic with error handling in `internal/controller/gcp/service/chain.go`

**Checkpoint**: Foundation ready - handler infrastructure complete, user story implementation can now begin

---

## Phase 3: User Story 1 - Refactor Controller Reconciliation Logic (Priority: P1) 🎯 MVP

**Goal**: Refactor the monolithic Reconcile function into a Chain of Responsibility pattern with discrete handlers for each reconciliation step

**Independent Test**: Run existing test suite and verify refactored controller produces identical reconciliation results to current implementation. Verify handlers execute in correct order through logging.

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T010 [US1] Write unit test for Fetch handler in `internal/controller/gcp/service/handlers/fetch_handler_test.go` with mocked Kubernetes client
- [X] T011 [US1] Write unit test for Finalizer handler in `internal/controller/gcp/service/handlers/finalizer_handler_test.go` with mocked Kubernetes client
- [X] T012 [US1] Write unit test for Authentication handler in `internal/controller/gcp/service/handlers/auth_handler_test.go` with mocked Kubernetes client and GCP client factory
- [X] T013 [US1] Write unit test for Period Validation handler in `internal/controller/gcp/service/handlers/period_handler_test.go` with mocked period utilities
- [X] T014 [US1] Write unit test for Resource Scaling handler in `internal/controller/gcp/service/handlers/scaling_handler_test.go` with mocked GCP client and resource utilities
- [X] T015 [US1] Write unit test for Status Update handler in `internal/controller/gcp/service/handlers/status_handler_test.go` with mocked Kubernetes client
- [X] T016 [US1] Write integration test for full handler chain execution in `internal/controller/gcp/service/chain_test.go` verifying handler order and context passing

### Implementation for User Story 1

- [X] T017 [US1] Implement Fetch handler in `internal/controller/gcp/service/handlers/fetch_handler.go` that fetches scaler resource from Kubernetes API
- [X] T018 [US1] Implement Finalizer handler in `internal/controller/gcp/service/handlers/finalizer_handler.go` that manages finalizer lifecycle (add/remove)
- [X] T019 [US1] Implement Authentication handler in `internal/controller/gcp/service/handlers/auth_handler.go` that sets up GCP client with authentication
- [X] T020 [US1] Implement Period Validation handler in `internal/controller/gcp/service/handlers/period_handler.go` that validates and determines current time period
- [X] T021 [US1] Implement Resource Scaling handler in `internal/controller/gcp/service/handlers/scaling_handler.go` that scales GCP resources based on period
- [X] T022 [US1] Implement Status Update handler in `internal/controller/gcp/service/handlers/status_handler.go` that updates scaler status with operation results
- [X] T023 [US1] Create handler chain constructor in `internal/controller/gcp/service/chain.go` that registers handlers in fixed order (fetch → finalizer → auth → period → scaling → status)
- [X] T024 [US1] Refactor Reconcile function in `internal/controller/gcp/scaler_controller.go` to delegate to handler chain instead of inline logic
- [X] T025 [US1] Update ScalerReconciler struct in `internal/controller/gcp/scaler_controller.go` to include chain instance with dependency injection

**Checkpoint**: At this point, User Story 1 should be complete - refactored controller uses handler chain and produces identical results to current implementation

---

## Phase 4: User Story 2 - Maintain Backward Compatibility (Priority: P1)

**Goal**: Ensure refactored controller maintains 100% backward compatibility with existing GCP scaler resources and test suite

**Independent Test**: Run existing test suite (`make test`) and verify all tests pass with identical results. Deploy refactored controller and verify existing GCP scaler resources continue to reconcile correctly.

### Tests for User Story 2

- [X] T026 [US2] Run existing unit tests in `internal/controller/gcp/scaler_controller_test.go` and verify all pass
- [X] T027 [US2] Run existing integration tests and verify all pass with identical results
- [X] T028 [US2] Create compatibility test in `internal/controller/gcp/compatibility_test.go` that verifies reconciliation results match current implementation for same input resources

### Implementation for User Story 2

- [X] T029 [US2] Verify handler chain produces identical ctrl.Result values as current implementation for all test cases
- [X] T030 [US2] Verify handler chain produces identical status updates as current implementation
- [X] T031 [US2] Verify handler chain handles all existing edge cases (run-once periods, finalizer cleanup, error scenarios)
- [X] T032 [US2] Update controller tests in `internal/controller/gcp/scaler_controller_test.go` to work with refactored implementation while maintaining same test assertions
- [X] T033 [US2] Verify API contracts remain unchanged - no modifications to Gcp CRD or resource specifications

**Checkpoint**: At this point, User Story 2 should be complete - all existing tests pass and backward compatibility is verified

---

## Phase 5: User Story 3 - Improve Code Testability (Priority: P2)

**Goal**: Enable independent unit testing of each handler with mocked dependencies, improving test coverage and reducing test complexity

**Independent Test**: Verify each handler can be unit tested independently with mocked dependencies. Measure test execution time - each handler test should execute in under 100ms. Verify test setup complexity is reduced by at least 60% compared to integration tests.

### Tests for User Story 3

- [X] T034 [US3] Verify Fetch handler unit test executes in under 100ms in `internal/controller/gcp/service/handlers/fetch_handler_test.go`
- [X] T035 [US3] Verify Finalizer handler unit test executes in under 100ms in `internal/controller/gcp/service/handlers/finalizer_handler_test.go`
- [X] T036 [US3] Verify Authentication handler unit test executes in under 100ms in `internal/controller/gcp/service/handlers/auth_handler_test.go`
- [X] T037 [US3] Verify Period Validation handler unit test executes in under 100ms in `internal/controller/gcp/service/handlers/period_handler_test.go`
- [X] T038 [US3] Verify Resource Scaling handler unit test executes in under 100ms in `internal/controller/gcp/service/handlers/scaling_handler_test.go`
- [X] T039 [US3] Verify Status Update handler unit test executes in under 100ms in `internal/controller/gcp/service/handlers/status_handler_test.go`
- [X] T040 [US3] Measure test setup complexity reduction: compare handler unit test setup (mocked dependencies) vs integration test setup (real Kubernetes/GCP clients)

### Implementation for User Story 3

- [X] T041 [US3] Create mock interfaces for all handler dependencies (Kubernetes client, GCP client, logger) in `internal/controller/gcp/service/mocks/` or using existing mock infrastructure
- [X] T042 [US3] Ensure all handlers use dependency injection via constructors (no global state) in handler implementations
- [X] T043 [US3] Verify each handler test uses only mocked dependencies (no Kubernetes or GCP API access required)
- [X] T044 [US3] Run test coverage analysis: `go test -coverprofile=coverage.out ./internal/controller/gcp/service/...` and verify coverage reaches at least 80%
- [X] T045 [US3] Document handler testing patterns in `internal/controller/gcp/service/handlers/README.md` for future handler development

**Checkpoint**: At this point, User Story 3 should be complete - all handlers have independent unit tests with mocked dependencies, test execution is fast, and coverage meets 80% threshold

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T046 [P] Verify cyclomatic complexity reduction: measure complexity of refactored Reconcile function vs current implementation, verify at least 50% reduction (SC-002)
- [X] T047 [P] Run full test suite and verify all tests pass: `make test`
- [X] T048 [P] Verify test coverage meets 80% threshold: `make test-coverage` and check coverage percentage (SC-003)
- [X] T049 [P] Update controller documentation in `internal/controller/gcp/README.md` explaining handler chain architecture
- [X] T050 [P] Add structured logging to handler chain execution in `internal/controller/gcp/service/chain.go` with handler execution start/end, errors, requeue requests
- [X] T051 [P] Verify error categorization works correctly: test critical errors stop chain, recoverable errors allow continuation with requeue
- [X] T052 [P] Verify handler skipping works correctly: test handler can skip remaining handlers (e.g., "no action" period scenario)
- [X] T053 [P] Verify context modification works correctly: test later handlers overwrite earlier changes (last write wins)
- [X] T054 [P] Verify requeue behavior works correctly: test first handler's requeue delay takes precedence
- [X] T055 [P] Code cleanup: remove any temporary code or debug statements used during refactoring
- [X] T056 [P] Run quickstart.md validation: verify all examples in quickstart.md work correctly with refactored implementation
- [X] T057 [P] Demonstrate extensibility: Add example/test handler to chain without modifying existing handlers (e.g., validation handler, logging handler) to verify SC-006 extensibility requirement

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P1 → P2)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories. Implements core handler chain infrastructure.
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - Depends on US1 (needs handler chain to exist for compatibility testing)
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Depends on US1 (needs handlers to exist for unit testing)

### Within Each User Story

- Tests MUST be written and FAIL before implementation (TDD approach)
- Handler implementations before chain integration
- Chain integration before controller refactoring
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel (T002, T003)
- All Foundational tasks marked [P] can run in parallel (T005, T006, T007)
- Within US1: Handler implementations (T017-T022) can be developed in parallel after tests are written
- Within US1: Handler tests (T010-T016) can be written in parallel
- All Polish tasks marked [P] can run in parallel

---

## Parallel Example: User Story 1

```bash
# Handler tests that can run in parallel:
Task: "Write unit test for Fetch handler in internal/controller/gcp/service/handlers/fetch_handler_test.go"
Task: "Write unit test for Finalizer handler in internal/controller/gcp/service/handlers/finalizer_handler_test.go"
Task: "Write unit test for Authentication handler in internal/controller/gcp/service/handlers/auth_handler_test.go"

# Handler implementations that can run in parallel (after tests):
Task: "Implement Fetch handler in internal/controller/gcp/service/handlers/fetch_handler.go"
Task: "Implement Finalizer handler in internal/controller/gcp/service/handlers/finalizer_handler.go"
Task: "Implement Authentication handler in internal/controller/gcp/service/handlers/auth_handler.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Handler chain refactoring)
4. **STOP and VALIDATE**: Test refactored controller produces identical results
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP - refactored controller!)
3. Add User Story 2 → Test independently → Deploy/Demo (backward compatibility verified)
4. Add User Story 3 → Test independently → Deploy/Demo (improved testability)
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (handler implementations)
   - Developer B: User Story 1 (handler tests) → then User Story 2 (compatibility)
   - Developer C: User Story 3 (testability improvements)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Tests MUST be written first and fail before implementation (TDD)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
- Handler implementations follow Clean Architecture: business logic in service layer, interfaces for all dependencies
- All handlers use dependency injection via constructors (no global state)
