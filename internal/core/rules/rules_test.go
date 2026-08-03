package rules

import (
	"reflect"
	"strings"
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

func TestStripComments(t *testing.T) {
	input := `
		// line comment
		val x = 10 /* block comment */
		val s = "string with // comments inside"
	`
	got := stripComments(input)
	if !containsWord(got, "x") {
		t.Error("should preserve code variables")
	}
	if stringsContainsAny(got, "line comment", "block comment") {
		t.Error("should strip comments")
	}
	if !stringsContainsAny(got, "string with // comments inside") {
		t.Error("should preserve string content")
	}
}

func TestStripCommentsAndStrings(t *testing.T) {
	input := `
		// line comment
		val x = "string literal" /* block comment */
	`
	got := stripCommentsAndStrings(input)
	if !containsWord(got, "x") {
		t.Error("should preserve code variables")
	}
	if stringsContainsAny(got, "line comment", "string literal", "block comment") {
		t.Error("should strip comments and string literals")
	}
}

func TestExtractArguments(t *testing.T) {
	cases := []struct {
		input    string
		keyword  string
		expected string
	}{
		{
			input:    `items(list, key = { it.id }) { ... }`,
			keyword:  "items",
			expected: "list, key = { it.id }",
		},
		{
			input:    `Log.d("Tag", "msg: " + email)`,
			keyword:  "Log.d",
			expected: `"Tag", "msg: " + email`,
		},
		{
			input:    `itemsIndexed(list, key = { _, item -> item.id })`,
			keyword:  "itemsIndexed",
			expected: "list, key = { _, item -> item.id }",
		},
	}

	for _, tc := range cases {
		idx := findKeywordIndex(tc.input, tc.keyword)
		if idx == -1 {
			t.Fatalf("keyword %s not found in input %s", tc.keyword, tc.input)
		}
		args, endIdx := extractArguments(tc.input, idx+len(tc.keyword))
		if endIdx == -1 {
			t.Fatalf("could not extract arguments for %s", tc.keyword)
		}
		if args != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, args)
		}
	}
}

func TestComposeMissingKeyDetector(t *testing.T) {
	rule := types.Rule{ID: "compose-missing-key", Cluster: "compose-performance", Severity: types.SeverityError}
	det := &ComposeMissingKeyDetector{rule: rule}

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name: "items without key",
			content: `
				LazyColumn {
					items(itemsList) { item -> Text(item.name) }
				}
			`,
			want: 1,
		},
		{
			name: "items with key parameter",
			content: `
				LazyColumn {
					items(itemsList, key = { it.id }) { item -> Text(item.name) }
				}
			`,
			want: 0,
		},
		{
			name: "itemsIndexed without key",
			content: `
				LazyColumn {
					itemsIndexed(itemsList) { idx, item -> Text(item.name) }
				}
			`,
			want: 1,
		},
		{
			name: "itemsIndexed with key",
			content: `
				LazyColumn {
					itemsIndexed(itemsList, key = { idx, item -> item.id }) { idx, item -> Text(item.name) }
				}
			`,
			want: 0,
		},
		{
			name: "items inside comments should be ignored",
			content: `
				// items(itemsList) { ... }
				/* items(itemsList) */
			`,
			want: 0,
		},
		{
			name: "items inside string literal should be ignored",
			content: `
				val s = "items(itemsList) should be checked"
			`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commStr := stripComments(tt.content)
			commStringStr := stripCommentsAndStrings(tt.content)
			findings := det.Check("File.kt", tt.content, commStr, commStringStr)
			if len(findings) != tt.want {
				t.Errorf("Check() got %d findings, want %d", len(findings), tt.want)
			}
		})
	}
}

func TestSecLogPiiDetector(t *testing.T) {
	rule := types.Rule{ID: "sec-log-pii", Cluster: "security", Severity: types.SeverityError}
	det := &SecLogPiiDetector{rule: rule}

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "logging email address",
			content: `Log.d("User", "user email: admin@domain.com")`,
			want:    1,
		},
		{
			name:    "logging password variable",
			content: `Log.w(TAG, "password is: $password")`,
			want:    1,
		},
		{
			name:    "logging token key",
			content: `Timber.e(t, "Auth token expired: $token")`,
			want:    1,
		},
		{
			name:    "harmless logging",
			content: `Log.i("Network", "Request completed successfully in 200ms")`,
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commStr := stripComments(tt.content)
			commStringStr := stripCommentsAndStrings(tt.content)
			findings := det.Check("File.kt", tt.content, commStr, commStringStr)
			if len(findings) != tt.want {
				t.Errorf("Check() got %d findings, want %d", len(findings), tt.want)
			}
		})
	}
}

func TestSecWebViewJavascriptEnabledDetector(t *testing.T) {
	rule := types.Rule{ID: "sec-webview-javascript-enabled", Cluster: "security", Severity: types.SeverityError}
	det := &SecWebViewJavascriptEnabledDetector{rule: rule}

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "WebView enabling JS settings object",
			content: `webView.settings.javaScriptEnabled = true`,
			want:    1,
		},
		{
			name:    "WebView enabling JS direct settings",
			content: `settings.javaScriptEnabled = true`,
			want:    1,
		},
		{
			name:    "WebView setting JS to false",
			content: `settings.javaScriptEnabled = false`,
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commStr := stripComments(tt.content)
			commStringStr := stripCommentsAndStrings(tt.content)
			findings := det.Check("File.kt", tt.content, commStr, commStringStr)
			if len(findings) != tt.want {
				t.Errorf("Check() got %d findings, want %d", len(findings), tt.want)
			}
		})
	}
}

func TestCoroutineDispatchersHardcodedDetector(t *testing.T) {
	rule := types.Rule{ID: "coroutine-dispatchers-hardcoded", Cluster: "coroutines", Severity: types.SeverityInfo}
	det := &CoroutineDispatchersHardcodedDetector{rule: rule}

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "hardcoded IO dispatcher",
			content: `withContext(Dispatchers.IO) { ... }`,
			want:    1,
		},
		{
			name:    "hardcoded Default dispatcher",
			content: `val scope = CoroutineScope(Dispatchers.Default)`,
			want:    1,
		},
		{
			name: "dispatchers injected",
			content: `
				class MyRepository(private val ioDispatcher: CoroutineDispatcher = Dispatchers.IO)
			`,
			want: 1, // hardcoded default parameter value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := det.Check("Repository.kt", tt.content, stripComments(tt.content), stripCommentsAndStrings(tt.content))
			if len(got) != tt.want {
				t.Errorf("got %d findings, want %d", len(got), tt.want)
			}
		})
	}
}

func TestArchPresentationDependsOnDataDetector(t *testing.T) {
	rule := types.Rule{ID: "arch-presentation-depends-on-data", Cluster: "architecture", Severity: types.SeverityError}
	det := &ArchPresentationDependsOnDataDetector{rule: rule}

	content := `
		package com.example.app.ui
		import com.example.app.data.UserRepositoryImpl

		@Composable
		fun UserScreen() {
			val repo = UserRepositoryImpl()
		}
	`
	got := det.Check("presentation/UserScreen.kt", content, stripComments(content), stripCommentsAndStrings(content))
	if len(got) < 1 {
		t.Errorf("expected presentation layer error when importing/referencing data impl, got %d findings", len(got))
	}
}

func TestArchViewModelContractDetector(t *testing.T) {
	rule := types.Rule{ID: "arch-viewmodel-contract", Cluster: "architecture", Severity: types.SeverityWarning}
	det := &ArchViewModelContractDetector{rule: rule}

	content := `
		class UserViewModel(private val repo: UserRepositoryImpl) : ViewModel()
	`
	got := det.Check("UserViewModel.kt", content, stripComments(content), stripCommentsAndStrings(content))
	if len(got) != 1 {
		t.Errorf("expected 1 finding for ViewModel receiving RepositoryImpl, got %d", len(got))
	}
}

func TestArchUseCaseContractDetector(t *testing.T) {
	rule := types.Rule{ID: "arch-usecase-contract", Cluster: "architecture", Severity: types.SeverityWarning}
	det := &ArchUseCaseContractDetector{rule: rule}

	content := `
		class GetUserUseCase(private val dataSource: UserDataSource)
	`
	got := det.Check("domain/usecase/GetUserUseCase.kt", content, stripComments(content), stripCommentsAndStrings(content))
	if len(got) != 1 {
		t.Errorf("expected 1 finding for UseCase receiving DataSource, got %d", len(got))
	}
}

func TestArchUseCaseMultiplePublicMethodsDetector(t *testing.T) {
	rule := types.Rule{ID: "arch-usecase-multiple-public-methods", Cluster: "architecture", Severity: types.SeverityWarning}
	det := &ArchUseCaseMultiplePublicMethodsDetector{rule: rule}

	content := `
		class GetUserUseCase {
			fun execute() {}
			fun extraPublicMethod() {}
		}
	`
	got := det.Check("GetUserUseCase.kt", content, stripComments(content), stripCommentsAndStrings(content))
	if len(got) != 1 {
		t.Errorf("expected 1 finding for UseCase with multiple public methods, got %d", len(got))
	}
}

func TestArchMisplacedDomainLogicDetector(t *testing.T) {
	rule := types.Rule{ID: "arch-misplaced-domain-logic", Cluster: "architecture", Severity: types.SeverityWarning}
	det := &ArchMisplacedDomainLogicDetector{rule: rule}

	content := `
		class CheckoutViewModel : ViewModel() {
			fun calculateDiscount(price: Double): Double {
				return price * 0.1
			}
		}
	`
	got := det.Check("CheckoutViewModel.kt", content, stripComments(content), stripCommentsAndStrings(content))
	if len(got) != 1 {
		t.Errorf("expected 1 finding for business logic in ViewModel, got %d", len(got))
	}
}

func TestArchMisplacedDataLogicDetector(t *testing.T) {
	rule := types.Rule{ID: "arch-misplaced-data-logic", Cluster: "architecture", Severity: types.SeverityError}
	det := &ArchMisplacedDataLogicDetector{rule: rule}

	content := `
		class GetUserUseCase {
			fun execute() {
				val client = HttpClient()
			}
		}
	`
	got := det.Check("GetUserUseCase.kt", content, stripComments(content), stripCommentsAndStrings(content))
	if len(got) != 1 {
		t.Errorf("expected 1 finding for HttpClient in UseCase, got %d", len(got))
	}
}

func TestArchModelMappingLeakDetector(t *testing.T) {
	rule := types.Rule{ID: "arch-model-mapping-leak", Cluster: "architecture", Severity: types.SeverityWarning}
	det := &ArchModelMappingLeakDetector{rule: rule}

	content := `
		@Composable
		fun UserCard(user: UserResponse) {}
	`
	got := det.Check("ui/UserCard.kt", content, stripComments(content), stripCommentsAndStrings(content))
	if len(got) != 1 {
		t.Errorf("expected 1 finding for UserResponse DTO in Presentation, got %d", len(got))
	}
}

func TestErrorHandlingLayerMappingDetector(t *testing.T) {
	rule := types.Rule{ID: "error-handling-layer-mapping", Cluster: "error-handling", Severity: types.SeverityWarning}
	det := &ErrorHandlingLayerMappingDetector{rule: rule}

	content := `
		fun fetchData() {
			try {
				doSomething()
			} catch (e: Exception) {
				println(e)
			}
		}
	`
	got := det.Check("UserScreen.kt", content, stripComments(content), stripCommentsAndStrings(content))
	if len(got) != 1 {
		t.Errorf("expected 1 finding for raw generic catch, got %d", len(got))
	}
}

func TestArchViewModelMviSuggestionDetector(t *testing.T) {
	rule := types.Rule{ID: "arch-viewmodel-mvi-suggestion", Cluster: "architecture", Severity: types.SeverityInfo}
	det := &ArchViewModelMviSuggestionDetector{rule: rule}

	content := `
		class HomeViewModel : ViewModel() {
			val state1 = MutableStateFlow(0)
			val state2 = MutableStateFlow("")
			val state3 = MutableStateFlow(false)
		}
	`
	got := det.Check("HomeViewModel.kt", content, stripComments(content), stripCommentsAndStrings(content))
	if len(got) != 1 {
		t.Errorf("expected 1 finding suggesting MVI for 3 StateFlows, got %d", len(got))
	}
}

func TestUIHardcodedStringsDetector(t *testing.T) {
	rule := types.Rule{ID: "ui-hardcoded-strings", Cluster: "clean-code", Severity: types.SeverityInfo}
	det := &UIHardcodedStringsDetector{rule: rule}

	content := `
		@Composable
		fun LoginButton() {
			Text("Login Now")
		}
	`
	got := det.Check("ui/LoginButton.kt", content, stripComments(content), stripCommentsAndStrings(content))
	if len(got) != 1 {
		t.Errorf("expected 1 finding for hardcoded UI text, got %d", len(got))
	}
}

func TestTestabilityDirectInstantiationDetector(t *testing.T) {
	rule := types.Rule{ID: "testability-direct-instantiation", Cluster: "testing", Severity: types.SeverityError}
	det := &TestabilityDirectInstantiationDetector{rule: rule}

	content := `
		class UserViewModel : ViewModel() {
			val repo = UserRepositoryImpl()
		}
	`
	got := det.Check("UserViewModel.kt", content, stripComments(content), stripCommentsAndStrings(content))
	if len(got) != 1 {
		t.Errorf("expected 1 finding for direct instantiation of RepositoryImpl, got %d", len(got))
	}
}

// ----------------------------------------------------
// Test Helpers
// ----------------------------------------------------

func stringsContainsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if reflect.ValueOf(sub).IsValid() && reflect.ValueOf(s).IsValid() {
			if strings.Contains(s, sub) {
				return true
			}
		}
	}
	return false
}

func findKeywordIndex(content, keyword string) int {
	return strings.Index(content, keyword)
}
