# Tasks: Rewrite K8s Controller Using Chain of Responsibility Pattern

**Input**: Design documents from `/specs/002-k8s-controller-chain/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are explicitly required by the constitution (TDD for service layer) and feature specification (SC-003: 80% test coverage, SC-004: handler tests <100ms).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: Repository root structure
- Refactoring affects: `internal/controller/k8s/` and new `internal/controller/k8s/service/` directory

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure for handler chain infrastructure

- [X] T001 Create service directory structure: `internal/controller/k8s/service/` and `internal/controller/k8s/service/handlers/` directories
- [X] T002 [P] Verify Go 1.25.1 is available and matches go.mod
- [X] T003 [P] Verify existing dependencies are available: controller-runtime, zerolog, ginkgo/gomega

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Create Handler interface definition in `internal/controller/k8s/service/interfaces.go` with `execute(ctx *ReconciliationContext) error` and `setNext(next Handler)` methods
- [X] T005 [P] Create ReconciliationContext structure in `internal/controller/k8s/service/context.go` with all required fields (Request, Client, K8sClient, DynamicClient, Logger, Scaler, Secret, Period, ResourceConfig, SuccessResults, FailedResults, ShouldFinalize, SkipRemaining, RequeueAfter)
- [X] T006 [P] Create error categorization types (CriticalError, RecoverableError) in `internal/controller/k8s/service/errors.go` with helper functions `NewCriticalError()`, `NewRecoverableError()`, and `IsCriticalError()`

**Checkpoint**: Foundation ready - handler infrastructure complete, user story implementation can now begin

---

## Phase 3: User Story 1 - Refactor Controller Reconciliation Logic (Priority: P1) 🎯 MVP

**Goal**: Refactor the monolithic Reconcile function into a Chain of Responsibility pattern with discrete handlers for each reconciliation step using the classic refactoring.guru pattern

**Independent Test**: Run existing test suite and verify refactored controller produces identical reconciliation results to current implementation. Verify handlers execute in correct order through logging.

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T007 [US1] Write unit test for Fetch handler in `internal/controller/k8s/service/handlers/fetch_handler_test.go` with mocked Kubernetes client
- [X] T008 [US1] Write unit test for Finalizer handler in `internal/controller/k8s/service/handlers/finalizer_handler_test.go` with mocked Kubernetes client
- [X] T009 [US1] Write unit test for Authentication handler in `internal/controller/k8s/service/handlers/auth_handler_test.go` with mocked Kubernetes client and K8s client factory
- [X] T010 [US1] Write unit test for Period Validation handler in `internal/controller/k8s/service/handlers/period_handler_test.go` with mocked period utilities
- [X] T011 [US1] Write unit test for Resource Scaling handler in `internal/controller/k8s/service/handlers/scaling_handler_test.go` with mocked K8s client and resource utilities
- [X] T012 [US1] Write unit test for Status Update handler in `internal/controller/k8s/service/handlers/status_handler_test.go` with mocked Kubernetes client
- [X] T013 [US1] Write integration test for full handler chain execution in `internal/controller/k8s/service/chain_test.go` verifying handler order and context passing

### Implementation for User Story 1

- [X] T014 [US1] Implement Fetch handler in `internal/controller/k8s/service/handlers/fetch_handler.go` that fetches scaler resource from Kubernetes API and implements Handler interface with `execute()` and `setNext()` methods
- [X] T015 [US1] Implement Finalizer handler in `internal/controller/k8s/service/handlers/finalizer_handler.go` that manages finalizer lifecycle (add/remove) and implements Handler interface
- [X] T016 [US1] Implement Authentication handler in `internal/controller/k8s/service/handlers/auth_handler.go` that sets up K8s client with authentication and implements Handler interface
- [X] T017 [US1] Implement Period Validation handler in `internal/controller/k8s/service/handlers/period_handler.go` that validates and determines current time period and implements Handler interface
- [X] T018 [US1] Implement Resource Scaling handler in `internal/controller/k8s/service/handlers/scaling_handler.go` that scales K8s resources based on period and implements Handler interface
- [X] T019 [US1] Implement Status Update handler in `internal/controller/k8s/service/handlers/status_handler.go` that updates scaler status with operation results and implements Handler interface
- [X] T020 [US1] Create handler chain constructor in `internal/controller/k8s/scaler_controller.go` that creates handlers and links them via `setNext()` calls in fixed order (fetch → finalizer → auth → period → scaling → status)
- [X] T021 [US1] Refactor Reconcile function in `internal/controller/k8s/scaler_controller.go` to delegate to handler chain instead of inline logic
- [X] T022 [US1] Update ScalerReconciler struct in `internal/controller/k8s/scaler_controller.go` to include chain initialization method

**Checkpoint**: At this point, User Story 1 should be complete - refactored controller uses handler chain and produces identical results to current implementation

---

## Phase 4: User Story 2 - Maintain Backward Compatibility (Priority: P1)

**Goal**: Ensure refactored controller maintains 100% backward compatibility with existing K8s scaler resources and test suite

**Independent Test**: Run existing test suite (`make test`) and verify all tests pass with identical results. Deploy refactored controller and verify existing K8s scaler resources continue to reconcile correctly.

### Tests for User Story 2

- [X] T023 [US2] Run existing unit tests in `internal/controller/k8s/scaler_controller_test.go` and verify all pass
- [X] T024 [US2] Run existing integration tests and verify all pass with identical results
- [X] T025 [US2] Create compatibility test in `internal/controller/k8s/compatibility_test.go` that verifies reconciliation results match current implementation for same input resources

### Implementation for User Story 2

- [X] T026 [US2] Verify handler chain produces identical ctrl.Result values as current implementation for all test cases
- [X] T027 [US2] Verify handler chain produces identical status updates as current implementation
- [X] T028 [US2] Verify handler chain handles all existing edge cases (run-once periods, finalizer cleanup, error scenarios)
- [X] T029 [US2] Update controller tests in `internal/controller/k8s/scaler_controller_test.go` to work with refactored implementation while maintaining same test assertions
- [X] T030 [US2] Verify API contracts remain unchanged - no modifications to K8s CRD or resource specifications

**Checkpoint**: At this point, User Story 2 should be complete - all existing tests pass and backward compatibility is verified

---

## Phase 5: User Story 3 - Improve Code Testability (Priority: P2)

**Goal**: Enable independent unit testing of each handler with mocked dependencies, improving test coverage and reducing test complexity

**Independent Test**: Verify each handler can be unit tested independently with mocked dependencies. Measure test execution time - each handler test should execute in under 100ms. Verify test setup complexity is reduced by at least 60% compared to integration tests.

### Tests for User Story 3

- [X] T031 [US3] Verify Fetch handler unit test executes in under 100ms in `internal/controller/k8s/service/handlers/fetch_handler_test.go`
- [X] T032 [US3] Verify Finalizer handler unit test executes in under 100ms in `internal/controller/k8s/service/handlers/finalizer_handler_test.go`
- [X] T033 [US3] Verify Authentication handler unit test executes in under 100ms in `internal/controller/k8s/service/handlers/auth_handler_test.go`
- [X] T034 [US3] Verify Period Validation handler unit test executes in under 100ms in `internal/controller/k8s/service/handlers/period_handler_test.go`
- [X] T035 [US3] Verify Resource Scaling handler unit test executes in under 100ms in `internal/controller/k8s/service/handlers/scaling_handler_test.go`
- [X] T036 [US3] Verify Status Update handler unit test executes in under 100ms in `internal/controller/k8s/service/handlers/status_handler_test.go`
- [X] T037 [US3] Measure test setup complexity reduction: compare handler unit test setup (mocked dependencies) vs integration test setup (real Kubernetes/K8s clients)

### Implementation for User Story 3

- [X] T038 [US3] Create mock interfaces for all handler dependencies (Kubernetes client, K8s client factory, logger) in `internal/controller/k8s/service/mocks/` or using existing mock infrastructure
- [X] T039 [US3] Ensure all handlers use dependency injection via constructors (no global state) in handler implementations
- [X] T040 [US3] Verify each handler test uses only mocked dependencies (no Kubernetes or K8s API access required)
- [X] T041 [US3] Run test coverage analysis: `go test -coverprofile=coverage.out ./internal/controller/k8s/service/...` and verify coverage reaches at least 80%
- [X] T042 [US3] Document handler testing patterns in `internal/controller/k8s/service/handlers/README.md` for future handler development

**Checkpoint**: At this point, User Story 3 should be complete - all handlers have independent unit tests with mocked dependencies, test execution is fast, and coverage meets 80% threshold

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T043 [P] Verify cyclomatic complexity reduction: measure complexity of refactored Reconcile function vs current implementation, verify at least 50% reduction (SC-002)
- [X] T044 [P] Run full test suite and verify all tests pass: `make test`
- [X] T045 [P] Verify test coverage meets 80% threshold: `make test-coverage` and check coverage percentage (SC-003)
- [X] T046 [P] Update controller documentation in `internal/controller/k8s/README.md` explaining handler chain architecture
- [X] T047 [P] Add structured logging to handler chain execution with handler execution start/end, errors, requeue requests
- [X] T048 [P] Verify error categorization works correctly: test critical errors stop chain, recoverable errors allow continuation with requeue
- [X] T049 [P] Verify handler skipping works correctly: test handler can skip remaining handlers (e.g., "no action" period scenario)
- [X] T050 [P] Verify context modification works correctly: test later handlers overwrite earlier changes (last write wins)
- [X] T051 [P] Verify requeue behavior works correctly: test first handler's requeue delay takes precedence
- [X] T052 [P] Code cleanup: remove any temporary code or debug statements used during refactoring
- [X] T053 [P] Run quickstart.md validation: verify all examples in quickstart.md work correctly with refactored implementation
- [X] T054 [P] Demonstrate extensibility: Add example/test handler to chain without modifying existing handlers (e.g., validation handler, logging handler) to verify SC-006 extensibility requirement

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

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - Depends on User Story 1 completion (needs refactored controller to test compatibility)
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Depends on User Story 1 completion (needs handlers to test independently)

### Within Each User Story

- Tests (TDD) MUST be written and FAIL before implementation
- Handler implementations follow test-first approach
- Chain construction after all handlers are implemented
- Controller refactoring after chain is complete
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, User Story 1 can start
- All handler tests for User Story 1 marked [P] can run in parallel
- All handler implementations for User Story 1 can be worked on in parallel (different files)
- User Stories 2 and 3 can start after User Story 1 completes
- All Polish tasks marked [P] can run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch all handler tests for User Story 1 together:
Task: "Write unit test for Fetch handler in internal/controller/k8s/service/handlers/fetch_handler_test.go"
Task: "Write unit test for Finalizer handler in internal/controller/k8s/service/handlers/finalizer_handler_test.go"
Task: "Write unit test for Authentication handler in internal/controller/k8s/service/handlers/auth_handler_test.go"
Task: "Write unit test for Period Validation handler in internal/controller/k8s/service/handlers/period_handler_test.go"
Task: "Write unit test for Resource Scaling handler in internal/controller/k8s/service/handlers/scaling_handler_test.go"
Task: "Write unit test for Status Update handler in internal/controller/k8s/service/handlers/status_handler_test.go"

# Launch all handler implementations for User Story 1 together (after tests):
Task: "Implement Fetch handler in internal/controller/k8s/service/handlers/fetch_handler.go"
Task: "Implement Finalizer handler in internal/controller/k8s/service/handlers/finalizer_handler.go"
Task: "Implement Authentication handler in internal/controller/k8s/service/handlers/auth_handler.go"
Task: "Implement Period Validation handler in internal/controller/k8s/service/handlers/period_handler.go"
Task: "Implement Resource Scaling handler in internal/controller/k8s/service/handlers/scaling_handler.go"
Task: "Implement Status Update handler in internal/controller/k8s/service/handlers/status_handler.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Refactor Controller Reconciliation Logic)
4. **STOP and VALIDATE**: Test User Story 1 independently - verify refactored controller produces identical results
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Verify identical reconciliation results (MVP!)
3. Add User Story 2 → Test independently → Verify backward compatibility → Deploy/Demo
4. Add User Story 3 → Test independently → Verify testability improvements → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 - Handler tests and implementations (can work on different handlers in parallel)
   - Developer B: Can help with User Story 1 handlers or wait for completion
3. After User Story 1 completes:
   - Developer A: User Story 2 - Compatibility testing
   - Developer B: User Story 3 - Testability improvements
4. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing (TDD approach)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Handler chain uses classic refactoring.guru pattern: handlers have `execute()` and `setNext()` methods
- Handlers are linked via `setNext()` calls during chain construction
- First handler in chain is called to start execution
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
