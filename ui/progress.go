package ui

import "github.com/AllenDang/giu"

type ProgressiveTask struct {
	label    string
	progress float32
}

func NewProgressTask(label string) *ProgressiveTask {
	pt := &ProgressiveTask{
		label: label,
	}
	tasks = append(tasks, pt)
	if dialog != nil {
		_ = dialog.Text(label)
	}
	return pt
}

func (p *ProgressiveTask) UpdateProgress(v float32, label ...string) {
	p.progress = v
	if len(label) > 0 {
		p.label = label[0]
		if dialog != nil {
			_ = dialog.Text(label[0])
		}
	}
	if window != nil {
		giu.Update()
	}
}

func (p *ProgressiveTask) Close() {
	p.progress = 1
}
