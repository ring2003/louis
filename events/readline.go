// Copyright 2017 Louis McCormack
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

package events

import (
	"fmt"

	bpf "github.com/iovisor/gobpf/bcc"
)

type Readline struct{
	eventBase
	Str [80]byte
}

func (e Readline) Print() string {
	return fmt.Sprintf("%s", ReadCString(e.Str[:]))
}

func ReadlineBPF(evChan chan Event, ctx Ctx) {
	eventType := "readline"
	m := bpf.NewModule(`
		#include <uapi/linux/ptrace.h>

struct readline_event_t {
		u32 uid;
        u32 pid;
        char str[80];
} __attribute__((packed));

BPF_PERF_OUTPUT(readline_events);

int get_return_value(struct pt_regs *ctx) {
        struct readline_event_t event = {};
        if (!PT_REGS_RC(ctx))
            return 0;

		event.pid = bpf_get_current_pid_tgid();
		event.uid = bpf_get_current_uid_gid();
        bpf_probe_read(&event.str, sizeof(event.str), (void *)PT_REGS_RC(ctx));
        readline_events.perf_submit(ctx, &event, sizeof(event));

        return 0;
}`, []string{})
	defer m.Close()

	readlineUretprobe, err := m.LoadUprobe("get_return_value")
	if err != nil {
		ctx.Error <- "readline: failed to load get_return_value: " + err.Error()
		return
	}

	err = m.AttachUretprobe("/bin/bash", "readline", readlineUretprobe, -1)
	if err != nil {
		ctx.Error <- "readline: failed to attach return_value: " + err.Error()
		return
	}

	var event Readline
	readEvents(event, evChan, ctx, m, "readline_events", eventType)
}
