package bot

import (
	"crypto/tls"
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	irc "github.com/thoj/go-ircevent"

	"github.com/GinjaNinja32/DisGoIRC/format"
)

// IRCConfig represents the required configuration to connect to IRC
type IRCConfig struct {
	Nick string `json:"nick"`
	User string `json:"user"`
	Pass string `json:"pass"`

	SSL       bool   `json:"ssl"`
	SSLVerify bool   `json:"ssl_verify"`
	Server    string `json:"server"`

	CommandChars string `json:"command_chars"`
}

var (
	iSession *irc.Connection
)

func iInit() {
	c := conf.IRC
	iSession = irc.IRC(c.Nick, c.User)

	iSession.UseTLS = c.SSL
	// InsecureSkipVerify may be required to communicate with IRC servers.
	if !c.SSLVerify {
		iSession.TLSConfig = &tls.Config{InsecureSkipVerify: true} // nolint: gosec
	}
	iSession.Password = c.Pass
	iSession.AddCallback("PRIVMSG", iPrivmsg)
	iSession.AddCallback("CTCP_ACTION", iAction)

	err := iSession.Connect(c.Server)
	if err != nil {
		log.Fatalf("Failed to initialise IRC session: %s", err)
	}

	iSession.AddCallback("001", iSetupSession)

	go iHandleErrors()

	log.Infof("Connected to IRC")
}

func iHandleErrors() {
	for err := range iSession.ErrorChan() {
		log.Errorf("IRC error: %s", err) // TODO
	}
}

func iSetupSession(e *irc.Event) {
	for c := range conf.Mapping {
		iSession.Join(c)
	}
}

func iPrivmsg(e *irc.Event) {
	incomingIRC(e.Nick, strings.ToLower(e.Arguments[0]), e.Message())
}
func iAction(e *irc.Event) {
	incomingIRC(e.Nick, strings.ToLower(e.Arguments[0]), fmt.Sprintf("\x1d%s\x1d", e.Message()))
}

var outgoingNickRegex = regexp.MustCompile(`\b[a-zA-Z0-9]`)

// iAddAntiPing prefixes a message with a \uFEFF character to avoid pinging the user
func iAddAntiPing(s string) string {
	return outgoingNickRegex.ReplaceAllString(s, "$0\ufeff")
}

// iOutgoing transmits an IRC message prefixed with the provided nick if not set to anonymous
func iOutgoing(nick, channel string, message format.FormattedString, anonymous bool) {
	outgoingMessage := ""
	if anonymous {
		outgoingMessage = message.RenderIRC()
	} else {
		nick = iAddAntiPing(nick)
		outgoingMessage = fmt.Sprintf("<%s> %s", nick, message.RenderIRC())
	}
	iSession.Privmsg(channel, outgoingMessage)
}
