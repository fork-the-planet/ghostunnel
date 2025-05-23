/*-
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package policy

import (
	"context"
	"log"
	"os"
	"runtime"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/compile"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/stretchr/testify/assert"
)

var allowFoobarPolicyV0 string = `
package policy
import input
default allow := false
allow {
    input.name == "foobar"
}
`

var allowFoobarPolicyV1 string = `
package policy
import input
default allow := false
allow if {
    input.name == "foobar"
}
`

var allowAllPolicy string = `
package policy
import input
default allow := true
`

func TestPolicyInitFail(t *testing.T) {
	p, err := LoadFromPath("invalid", "invalid")
	assert.Nil(t, p, "policy should be nil")
	assert.NotNil(t, err, "error should not be nil")
}

func TestPolicyReloadFail(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Skip on Windows due to temp file handling issues
		t.Skip()
	}

	f, err := os.CreateTemp("", "policy*.rego")
	assert.Nil(t, err, "temp file error")
	defer os.Remove(f.Name())

	_, err = f.WriteAt([]byte(allowFoobarPolicyV0), 0)
	_ = f.Sync()
	assert.Nil(t, err, "temp file write error")

	p, err := LoadFromPath(f.Name(), "data.policy.allow")
	assert.NotNil(t, p, "policy was unexpectedly nil")
	assert.Nil(t, err, "error loading policy")
	if err != nil {
		t.Fatal(err)
	}

	// Remove + reload to test failure
	os.Remove(f.Name())
	err = p.Reload()
	assert.NotNil(t, err, "error should not be nil")
}

func TestPolicyReloading(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Skip on Windows due to temp file handling issues
		t.Skip()
	}

	f, err := os.CreateTemp("", "policy*.rego")
	assert.Nil(t, err, "temp file error")
	defer os.Remove(f.Name())

	_, err = f.WriteAt([]byte(allowFoobarPolicyV0), 0)
	_ = f.Sync()
	assert.Nil(t, err, "temp file write error")

	p, err := LoadFromPath(f.Name(), "data.policy.allow")
	assert.NotNil(t, p, "policy was unexpectedly nil")
	assert.Nil(t, err, "error loading policy")
	if err != nil {
		t.Fatal(err)
	}

	input := map[string]interface{}{"name": "foobar"}
	results, err := p.Eval(context.Background(), rego.EvalInput(input))
	assert.Nil(t, err, "error evaluating policy")
	if !results.Allowed() {
		log.Fatal("input foobar not allowed on original policy, though it should've been")
	}

	input = map[string]interface{}{"name": "barfoo"}
	results, err = p.Eval(context.Background(), rego.EvalInput(input))
	assert.Nil(t, err, "error evaluating policy")
	if results.Allowed() {
		log.Fatal("input barfoo allowed on original policy, though it should not have been")
	}

	_ = f.Truncate(0)
	_, err = f.WriteAt([]byte(allowAllPolicy), 0)
	_ = f.Sync()
	assert.Nil(t, err, "temp file write error")

	err = p.Reload()
	assert.Nil(t, err, "error reloading policy")

	input = map[string]interface{}{"name": "foobar"}
	results, err = p.Eval(context.Background(), rego.EvalInput(input))
	assert.Nil(t, err, "error evaluating policy")
	if !results.Allowed() {
		log.Fatal("input foobar not allowed on updated policy, though it should've been")
	}

	input = map[string]interface{}{"name": "barfoo"}
	results, err = p.Eval(context.Background(), rego.EvalInput(input))
	assert.Nil(t, err, "error evaluating policy")
	if !results.Allowed() {
		log.Fatal("input barfoo not allowed on updated policy, though it should've been")
	}
}

func TestPolicyBundleLoading(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Skip on Windows due to temp file handling issues
		t.Skip()
	}

	policy, err := os.CreateTemp("", "policy*.rego")
	assert.Nil(t, err, "temp file error")
	defer os.Remove(policy.Name())

	bundle, err := os.CreateTemp("", "policy-bundle*.tar.gz")
	assert.Nil(t, err, "temp file error")
	defer os.Remove(bundle.Name())

	_, err = policy.WriteAt([]byte(allowFoobarPolicyV1), 0)
	_ = policy.Sync()
	assert.Nil(t, err, "temp file write error")

	err = compile.New().
		WithPaths(policy.Name()).
		WithOutput(bundle).
		WithRegoVersion(ast.RegoV1).
		Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	p, err := LoadFromPath(bundle.Name(), "data.policy.allow")
	assert.NotNil(t, p, "policy was unexpectedly nil")
	assert.Nil(t, err, "error loading policy")
	if err != nil {
		t.Fatal(err)
	}

	input := map[string]interface{}{"name": "foobar"}
	results, err := p.Eval(context.Background(), rego.EvalInput(input))
	assert.Nil(t, err, "error evaluating policy")
	if !results.Allowed() {
		log.Fatal("input foobar not allowed on original policy, though it should've been")
	}

	input = map[string]interface{}{"name": "barfoo"}
	results, err = p.Eval(context.Background(), rego.EvalInput(input))
	assert.Nil(t, err, "error evaluating policy")
	if results.Allowed() {
		log.Fatal("input barfoo allowed on original policy, though it should not have been")
	}
}
