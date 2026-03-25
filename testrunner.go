package main

import (
	"fmt"
	"strings"
)

// GenerateTestRunner emits a main() that runs all @test methods and reports results
func GenerateTestRunner(testMethods []string) string {
	if len(testMethods) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n// ── @test runner ─────────────────────────────────────────────\n")
	sb.WriteString("#include <iostream>\n")
	sb.WriteString("#include <functional>\n")
	sb.WriteString("#include <vector>\n")
	sb.WriteString("#include <string>\n\n")
	sb.WriteString("int main() {\n")
	sb.WriteString("    using TestCase = std::pair<std::string, std::function<void()>>;\n")
	sb.WriteString("    std::vector<TestCase> tests = {\n")
	for _, m := range testMethods {
		// m is "ClassName::MethodName"
		parts := strings.SplitN(m, "::", 2)
		if len(parts) != 2 {
			continue
		}
		className, methodName := parts[0], parts[1]
		sb.WriteString(fmt.Sprintf("        {\"%s\", []() { %s obj; obj.%s(); }},\n",
			m, className, methodName))
	}
	sb.WriteString("    };\n\n")
	sb.WriteString("    int passed = 0, failed = 0;\n")
	sb.WriteString("    std::cout << \"\\n\" << \"=== Running \" << tests.size() << \" test(s) ===\" << \"\\n\\n\";\n")
	sb.WriteString("    for (auto& [name, fn] : tests) {\n")
	sb.WriteString("        std::cout << \"  [ RUN ] \" << name << \"\\n\";\n")
	sb.WriteString("        try {\n")
	sb.WriteString("            fn();\n")
	sb.WriteString("            std::cout << \"  [ OK  ] \" << name << \"\\n\";\n")
	sb.WriteString("            ++passed;\n")
	sb.WriteString("        } catch (const std::exception& ex) {\n")
	sb.WriteString("            std::cerr << \"  [FAIL] \" << name << \": \" << ex.what() << \"\\n\";\n")
	sb.WriteString("            ++failed;\n")
	sb.WriteString("        } catch (...) {\n")
	sb.WriteString("            std::cerr << \"  [FAIL] \" << name << \": (unknown exception)\" << \"\\n\";\n")
	sb.WriteString("            ++failed;\n")
	sb.WriteString("        }\n")
	sb.WriteString("    }\n\n")
	sb.WriteString("    std::cout << \"\\n=== Results: \" << passed << \" passed, \" << failed << \" failed ===\\n\";\n")
	sb.WriteString("    return failed > 0 ? 1 : 0;\n")
	sb.WriteString("}\n")
	return sb.String()
}

// GenerateAssertHeader emits a simple assert helper usable in @test methods
func GenerateAssertHeader() string {
	return `
// ── Test assertion helpers ─────────────────────────────────────────
#include <stdexcept>
#include <sstream>

namespace Assert {
    inline void IsTrue(bool cond, const std::string& msg = "Assert.IsTrue failed") {
        if (!cond) throw std::runtime_error(msg);
    }
    inline void IsFalse(bool cond, const std::string& msg = "Assert.IsFalse failed") {
        if (cond) throw std::runtime_error(msg);
    }
    template<typename T>
    inline void AreEqual(const T& expected, const T& actual, const std::string& msg = "") {
        if (expected != actual) {
            std::ostringstream oss;
            oss << "Assert.AreEqual failed";
            if (!msg.empty()) oss << ": " << msg;
            throw std::runtime_error(oss.str());
        }
    }
    template<typename T>
    inline void AreNotEqual(const T& a, const T& b, const std::string& msg = "") {
        if (a == b) {
            std::ostringstream oss;
            oss << "Assert.AreNotEqual failed";
            if (!msg.empty()) oss << ": " << msg;
            throw std::runtime_error(oss.str());
        }
    }
    inline void IsNull(const void* ptr, const std::string& msg = "Assert.IsNull failed") {
        if (ptr != nullptr) throw std::runtime_error(msg);
    }
    inline void IsNotNull(const void* ptr, const std::string& msg = "Assert.IsNotNull failed") {
        if (ptr == nullptr) throw std::runtime_error(msg);
    }
    inline void Fail(const std::string& msg = "Assert.Fail") {
        throw std::runtime_error(msg);
    }
}
`
}
