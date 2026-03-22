# Testing

## Framework

- **Ginkgo v2 / Gomega** for BDD-style tests
- **envtest** for Kubernetes API simulation
- **fake.NewClientBuilder()** for unit tests with fake K8s clients

## Test Structure (Ginkgo BDD)

```go
var _ = Describe("FetchHandler", func() {
    var (
        handler  service.Handler
        reconCtx *service.ReconciliationContext
    )

    BeforeEach(func() { /* setup */ })

    Context("when the scaler resource exists", func() {
        It("should fetch and set the scaler in context", func() {
            err := handler.Execute(reconCtx)
            Expect(err).ToNot(HaveOccurred())
            Expect(reconCtx.Scaler).ToNot(BeNil())
        })
    })
})
```

## Key Patterns

- **Fake client**: `fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()`
- **No-op logger**: `zerolog.Nop()`
- **Scheme registration**: Register API types before creating fake clients
- **Mock handlers**: Implement `Handler` interface with function fields
- **Resource types**: Use valid `common.ResourceKind` constants, not arbitrary strings
- **Error type checks**: `Expect(service.IsCriticalError(err)).To(BeTrue())`
- **Performance benchmarks**: `testing.B` with `b.ResetTimer()` and `b.N`

## Test File Locations

- `internal/controller/*/service/handlers/*_test.go` - Handler unit tests
- `internal/controller/*_test.go` - Controller integration tests
- `internal/controller/*/service/*_test.go` - Service layer tests
- `internal/webhook/v1alpha3/*_test.go` - Webhook tests
- `pkg/**/suite_test.go` - Package test suites
- `test/e2e/` - End-to-end tests with Kind cluster

## Requirements

- Mock external services via interfaces
- Table-driven tests for many input variants
- Benchmarks for performance regressions
- 80% minimum coverage (CI enforced)
- Test error classification (critical vs recoverable) explicitly
- Test handler chain order and next-handler invocation
- Test CRD conversions bidirectionally

## envtest Best Practices

- Start in `BeforeSuite`, stop in `AfterSuite`
- Separate namespaces per test
- Clean up in `AfterEach` or use unique names
- Reasonable timeouts for `Eventually`/`Consistently`
- Use `envtest.Environment.Config` for client configuration

## TDD Workflow (Service Layer)

1. Write tests first (Ginkgo/Gomega)
2. Get user approval
3. Run tests - verify they fail (`make test`)
4. Implement minimal code to pass
5. Refactor while keeping tests green
6. Verify 80%+ coverage (`make test-coverage`)
7. Lint (`make lint`)
