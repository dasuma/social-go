package utlis

import (
	"log"
	"strconv"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/mailbox"
	"github.com/AsynkronIT/protoactor-go/persistence"
)

type Provider struct {
	providerState persistence.ProviderState
}

var listContacts []interface{}

//-------------persistence

func NewProvider(snapshotInterval int) *Provider {
	return &Provider{
		providerState: persistence.NewInMemoryProvider(snapshotInterval),
	}
}

func (p *Provider) InitState(actorName string, eventNum, eventIndexAfterSnapshot int) {
	for i := 0; i < eventNum; i++ {
		p.providerState.PersistEvent(
			actorName,
			i,
			&Message{protoMsg: protoMsg{state: "state" + strconv.Itoa(i)}},
		)
	}
	p.providerState.PersistSnapshot(
		actorName,
		eventIndexAfterSnapshot,
		&Snapshot{protoMsg: protoMsg{state: "state" + strconv.Itoa(eventIndexAfterSnapshot-1)}},
	)
}

func (p *Provider) GetState() persistence.ProviderState {
	return p.providerState
}

type protoMsg struct{ state string }

func (p *protoMsg) Reset()         {}
func (p *protoMsg) String() string { return p.state }
func (p *protoMsg) ProtoMessage()  {}

type Message struct{ protoMsg }
type Snapshot struct{ protoMsg }

type Actor struct {
	persistence.Mixin
	state string
}

func (a *Actor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		log.Println("actor started")
	case *persistence.RequestSnapshot:
		log.Printf("snapshot internal state '%v'", a.state)
		a.PersistSnapshot(&Snapshot{protoMsg: protoMsg{state: a.state}})
	case *Snapshot:
		a.state = msg.state
		log.Printf("recovered from snapshot, internal state changed to '%v'", a.state)
	case *persistence.ReplayComplete:
		log.Printf("replay completed, internal state changed to '%v'", a.state)
	case *Message:
		scenario := "received replayed event"
		if !a.Recovering() {
			a.PersistReceive(msg)
			scenario = "received new message"
		}
		a.state = msg.state
		log.Printf("%s, internal state changed to '%v'\n", scenario, a.state)
	}
}

//---------end persisten
type mailboxLogger struct{}

func (m *mailboxLogger) MailboxStarted() {
	log.Print("Mailbox started")
}
func (m *mailboxLogger) MessagePosted(msg interface{}) {
	log.Printf("Message posted %v", msg)
}
func (m *mailboxLogger) MessageReceived(msg interface{}) {
	listContacts = append(listContacts, msg)
	log.Printf("Message received %v", msg)
	log.Printf("currentList %v", listContacts)
}
func (m *mailboxLogger) MailboxEmpty() {
	log.Print("No more messages")
}

type Contact struct {
	FirstName string
	Phone     string
	Id        int
}
type hello struct{ Who string }

var m = make(map[string]*actor.PID)

func createActor(contact Contact) {

	mm, ok := m[strconv.Itoa(contact.Id)]
	rootContext := actor.EmptyRootContext

	if !ok {

		provider := NewProvider(3)
		provider.InitState(strconv.Itoa(contact.Id), 4, 3)
		//props := actor.PropsFromProducer(func() actor.Actor { return &helloActor{} }).WithMailbox(mailbox.Unbounded(&mailboxLogger{})).WithReceiverMiddleware(persistence.Using(provider))
		props := actor.PropsFromProducer(func() actor.Actor { return &Actor{} }).WithMailbox(mailbox.Unbounded(&mailboxLogger{})).WithReceiverMiddleware(persistence.Using(provider))
		pid, err := rootContext.SpawnNamed(props, strconv.Itoa(contact.Id))

		if err != nil {
			log.Fatal("The actor name is already in use")
		}
		m[strconv.Itoa(contact.Id)] = pid
		rootContext.Send(pid, &hello{Who: "Roger"})
		rootContext.Send(pid, &contact)

	} else {
		rootContext.Send(mm, contact)
		rootContext.Send(mm, &contact)

		//rootContext.Send(pid, &hello{Who: "Roger"})
	}

	//rootContext.PoisonFuture(pid).Wait()
	//fmt.Printf("*** restart ***\n")

}
