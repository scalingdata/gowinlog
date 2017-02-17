// +build windows

package winlog

import (
	. "testing"
)

func TestWinlogWatcherConfiguresRendering(t *T) {
	watcher, err := NewWinLogWatcher()
	assertEqual(err, nil, t)

	watcher.SetRenderMessage(false)
	watcher.SetRenderLevel(false)
	watcher.SetRenderTask(false)
	watcher.SetRenderProvider(false)
	watcher.SetRenderOpcode(false)
	watcher.SetRenderChannel(false)
	watcher.SetRenderId(false)

	assertEqual(watcher.renderMessage, false, t)
	assertEqual(watcher.renderLevel, false, t)
	assertEqual(watcher.renderTask, false, t)
	assertEqual(watcher.renderProvider, false, t)
	assertEqual(watcher.renderOpcode, false, t)
	assertEqual(watcher.renderChannel, false, t)
	assertEqual(watcher.renderId, false, t)

	watcher.SetRenderMessage(true)
	watcher.SetRenderLevel(true)
	watcher.SetRenderTask(true)
	watcher.SetRenderProvider(true)
	watcher.SetRenderOpcode(true)
	watcher.SetRenderChannel(true)
	watcher.SetRenderId(true)

	assertEqual(watcher.renderMessage, true, t)
	assertEqual(watcher.renderLevel, true, t)
	assertEqual(watcher.renderTask, true, t)
	assertEqual(watcher.renderProvider, true, t)
	assertEqual(watcher.renderOpcode, true, t)
	assertEqual(watcher.renderChannel, true, t)
	assertEqual(watcher.renderId, true, t)

	watcher.Shutdown()
}
