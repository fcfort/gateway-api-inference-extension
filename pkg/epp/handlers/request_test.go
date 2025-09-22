/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handlers

import (
	"testing"

	configPb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extProcPb "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/metadata"
)

func TestHandleRequestHeaders(t *testing.T) {
	t.Parallel()

	// Setup a mock server and request context
	server := &StreamingServer{}

	reqCtx := &RequestContext{
		Request: &Request{
			Headers: make(map[string]string),
		},
	}

	req := &extProcPb.ProcessingRequest_RequestHeaders{
		RequestHeaders: &extProcPb.HttpHeaders{
			Headers: &configPb.HeaderMap{
				Headers: []*configPb.HeaderValue{
					{
						Key:   "x-test-header",
						Value: "test-value",
					},
					{
						Key:   metadata.FlowFairnessIDKey,
						Value: "test-fairness-id-value",
					},
				},
			},
			EndOfStream: false,
		},
	}

	err := server.HandleRequestHeaders(reqCtx, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if reqCtx.FairnessID != "test-fairness-id-value" {
		t.Errorf("expected fairness ID to be 'test-fairness-id-value', got %s", reqCtx.FairnessID)
	}
	if reqCtx.Request.Headers[metadata.FlowFairnessIDKey] == "test-fairness-id-value" {
		t.Errorf("expected fairness ID header to be removed from request headers, but it was not")
	}
}

func TestHandleRequestBody(t *testing.T) {
	s := &StreamingServer{}

	testCases := []struct {
		name                string
		jsonBody            string
		expectStreamOptions bool
		expectedPrompt      interface{}
	}{
		{
			name:                "stream true",
			jsonBody:            `{"prompt": "hello", "stream": true}`,
			expectStreamOptions: true,
			expectedPrompt:      "hello",
		},
		{
			name:                "stream false",
			jsonBody:            `{"prompt": "hello", "stream": false}`,
			expectStreamOptions: false,
			expectedPrompt:      "hello",
		},
		{
			name:                "stream not present",
			jsonBody:            `{"prompt": "hello"}`,
			expectStreamOptions: false,
			expectedPrompt:      "hello",
		},
		{
			name:                "empty body",
			jsonBody:            `{}`,
			expectStreamOptions: false,
			expectedPrompt:      nil,
		},
		{
			name:                "stream not a bool",
			jsonBody:            `{"prompt": "hello", "stream": "true"}`,
			expectStreamOptions: false,
			expectedPrompt:      "hello",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqCtx := &RequestContext{
				Request: &Request{
					Headers: make(map[string]string),
				},
			}

			err := s.HandleRequestBody(reqCtx, []byte(tc.jsonBody))
			require.NoError(t, err)

			bodyJSON := reqCtx.Request.Body

			if tc.expectStreamOptions {
				assert.Contains(t, bodyJSON, "stream_options")
				streamOptions, ok := bodyJSON["stream_options"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, true, streamOptions["include_usage"])
			} else {
				assert.NotContains(t, bodyJSON, "stream_options")
			}

			if tc.expectedPrompt != nil {
				assert.Equal(t, tc.expectedPrompt, bodyJSON["prompt"])
			}
		})
	}
}
