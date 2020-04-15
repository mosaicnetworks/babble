// Package proxy defines and implements AppProxy: the interface between Babble
// and an application.
//
// Babble communicates with the App through an AppProxy interface, which has two
// implementations:
//
// - SocketProxy: A SocketProxy connects to an App via TCP sockets. It enables
// the application to run in a separate process or machine, and to be written in
// any programming language.
//
// - InmemProxy: An InmemProxy uses native callback handlers to integrate Babble
// as a regular Go dependency.
package proxy
