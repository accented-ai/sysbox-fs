//
// Copyright 2026 Accented AI.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package fuse

import (
	"testing"

	bfuse "bazil.org/fuse"
	"github.com/stretchr/testify/require"
)

func TestConfigureOpenResponse(t *testing.T) {
	t.Parallel()

	t.Run("seekable", func(t *testing.T) {
		t.Parallel()

		resp := &bfuse.OpenResponse{Flags: bfuse.OpenKeepCache}
		configureOpenResponse(resp, false)

		require.Equal(t, bfuse.OpenKeepCache|bfuse.OpenDirectIO|openNoFlush, resp.Flags)
		require.Zero(t, resp.Flags&bfuse.OpenNonSeekable)
	})

	t.Run("non-seekable", func(t *testing.T) {
		t.Parallel()

		resp := &bfuse.OpenResponse{}
		configureOpenResponse(resp, true)

		require.Equal(t, bfuse.OpenDirectIO|bfuse.OpenNonSeekable|openNoFlush, resp.Flags)
	})
}
