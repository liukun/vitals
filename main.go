package main

import (
	"log"

	"github.com/egoist/mygo"
)

func main() {
	app := newApplication()
	mygo.App.SetName("Vitals")
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { app.quitting = true })
	mygo.App.OnNotificationClick(app.openHealthNotification)
	mygo.App.WhenReady(app.ready)
	mygo.App.OnWindowAllClosed(func() {})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
