package bot

import (
	"crypto/tls"
	"fmt"
	"regexp"

	log "github.com/Sirupsen/logrus"
	irc "github.com/thoj/go-ircevent"
)

type IRCConfig struct {
	Nick string "nick"
	User string "user"
	Pass string "pass"

	SSL    bool   "ssl"
	Server string "server"
}

var (
	iSession *irc.Connection
)

func iInit() {
	c := conf.IRC
	iSession = irc.IRC(c.Nick, c.User)

	iSession.UseTLS = c.SSL
	iSession.TLSConfig = &tls.Config{InsecureSkipVerify: true} // don't verify SSL certs
	iSession.Password = c.Pass
	iSession.AddCallback("PRIVMSG", iPrivmsg)
	iSession.AddCallback("CTCP_ACTION", iAction)

	err := iSession.Connect(c.Server)
	if err != nil {
		log.Fatalf("Failed to initialise IRC session: %s", err)
	}

	iSession.AddCallback("001", iSetupSession)

	log.Infof("Connected to IRC")
}

func iSetupSession(e *irc.Event) {
	for c := range conf.Mapping {
		iSession.Join(c)
	}
}

func iPrivmsg(e *irc.Event) {
	incomingIRC(e.Nick, e.Arguments[0], e.Message())
}
func iAction(e *irc.Event) {
	incomingIRC(e.Nick, e.Arguments[0], fmt.Sprintf("_%s_", e.Message()))
}

var outgoingNickRegex = regexp.MustCompile(`\b[a-zA-Z0-9]`)

func iOutgoing(nick, channel, message string) {
	// add a \uFEFF character to avoid pinging the user
	nick = outgoingNickRegex.ReplaceAllString(nick, "$0\ufeff")
	iSession.Privmsg(channel, fmt.Sprintf("<%s> %s", nick, message))
}