package notify

import (
	"context"
	"fmt"
	"os"
)

const DefaultBeepDevice = "/dev/ttyS1"

type beepNotifier struct{ device string }

// NewBeepNotifier returns a Notifier that triggers the Synology hardware
// buzzer by writing to device (typically /dev/ttyS1).
//
//	"2\n" → short beep  (import started / completed)
//	"3\n" → long beep   (import failed)
func NewBeepNotifier(device string) Notifier {
	return &beepNotifier{device: device}
}

func (b *beepNotifier) Notify(_ context.Context, event Event) error {
	var tone string
	switch event.Kind {
	case KindImportStarted, KindImportCompleted:
		tone = "2\n"
	case KindImportFailed:
		tone = "3\n"
	default:
		return nil
	}

	f, err := os.OpenFile(b.device, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("beep: open %s: %w", b.device, err)
	}
	defer f.Close()
	_, err = fmt.Fprint(f, tone)
	return err
}
