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

package seccomp

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestProxyOpenat2How(t *testing.T) {
	t.Parallel()

	flags := uint64(unix.O_PATH | unix.O_CLOEXEC | unix.O_NOFOLLOW)
	resolve := uint64(RESOLVE_NO_XDEV | RESOLVE_NO_MAGICLINKS | RESOLVE_NO_SYMLINKS | RESOLVE_BENEATH | RESOLVE_IN_ROOT | RESOLVE_CACHED)

	gotFlags, gotResolve := proxyOpenat2How(flags, resolve)

	require.Equal(t, flags&^uint64(unix.O_PATH), gotFlags)
	require.Equal(t, uint64(RESOLVE_IN_ROOT|RESOLVE_CACHED), gotResolve)
}
