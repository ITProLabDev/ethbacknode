package subscriptions

import (
	"encoding/json"

	"github.com/ITProLabDev/ethbacknode/clients/urpc"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// ServiceId is a unique identifier for a service subscription.
type ServiceId int

// NewSubscription creates a new subscription with the given service ID and endpoint URL.
// The fillSettings function allows customizing subscription settings.
func NewSubscription(serviceId ServiceId, endpointUrl string, fillSettings func(s *Subscription)) *Subscription {
	subscription := &Subscription{
		ServiceId:   serviceId,
		EndpointUrl: endpointUrl,
	}
	fillSettings(subscription)
	return subscription
}

// subscriptionsSave persists all subscriptions to storage.
func (s *Manager) subscriptionsSave() error {
	b, err := json.MarshalIndent(s.subscribers, "", "\t")
	if err != nil {
		return err
	}
	return s.subscribersStorage.Save(b)
}
// subscriptionsLoad loads subscriptions from storage.
func (s *Manager) subscriptionsLoad() error {
	s.subscribers = make(map[ServiceId]*Subscription)
	if !s.subscribersStorage.IsExists() {
		return s.subscriptionsColdStart()
	}
	b, err := s.subscribersStorage.Load()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.subscribers)
}

// subscriptionsColdStart initializes subscriptions with a default internal service.
func (s *Manager) subscriptionsColdStart() error {
	s.subscribers[0] = &Subscription{
		ServiceId: 0,
		Internal:  true,
	}
	b, _ := json.MarshalIndent(s.subscribers, "", "\t")
	return s.subscribersStorage.Save(b)
}

// subscriptionViewAll iterates over all subscriptions with a read lock.
func (s *Manager) subscriptionViewAll(viewver func(s *Subscription)) {
	s.subscribersMux.RLock()
	defer s.subscribersMux.RUnlock()
	for _, v := range s.subscribers {
		viewver(v)
	}
}

// SubscriptionGet retrieves a subscription by service ID.
// Returns ErrUnknownServiceId if not found.
func (s *Manager) SubscriptionGet(serviceId ServiceId) (subscription *Subscription, err error) {
	s.subscribersMux.RLock()
	defer s.subscribersMux.RUnlock()
	subscriptionMaster, found := s.subscribers[serviceId]
	if !found {
		return nil, ErrUnknownServiceId
	}
	subscription = &*subscriptionMaster
	return subscription, nil

}

// SubscriptionEdit modifies a subscription using the provided edit function.
// Persists changes after editing.
func (s *Manager) SubscriptionEdit(serviceId ServiceId, edit func(subscription *Subscription)) (err error) {
	s.subscribersMux.RLock()
	subscriptionMaster, found := s.subscribers[serviceId]
	s.subscribersMux.RUnlock()
	if !found {
		return ErrUnknownServiceId
	}
	//back:=subscriptionMaster.GetCopy()
	edit(subscriptionMaster)
	//TODO validate changes
	return s.subscriptionsSave()
}

// Subscription represents a service's subscription configuration.
// Controls what events to report and where to send notifications.
type Subscription struct {
	rpc                  *urpc.Client
	ServiceName          string          `json:"serviceName"`
	ServiceId            ServiceId       `json:"serviceId"`
	Internal             bool            `json:"internal,omitempty"`
	ApiToken             string          `json:"apiToken"`
	ApiKey               string          `json:"apiKey"`
	EndpointUrl          string          `json:"eventUrl"`
	ReportNewBlock       bool            `json:"reportNewBlock"`
	ReportIncomingTx     bool            `json:"reportIncomingTx"`
	ReportOutgoingTx     bool            `json:"reportOutgoingTx"`
	ReportMainCoin       bool            `json:"reportMainCoin"`
	ReportTokens         map[string]bool `json:"reportTokens"`
	ReportBalanceChange  bool            `json:"balanceChange"`
	GatherToMaster       bool            `json:"gatherToMaster"`
	MasterList           []string        `json:"masterList"`
	SecuritySignRequests bool            `json:"securitySignRequests,omitempty"`
	SecuritySignResponse bool            `json:"securitySignResponse,omitempty"`
	//Reserved for future use
	SecurityUseEncryption bool `json:"securityUseEncryption,omitempty"`
}

// equal compares two subscriptions for equality.
func (s *Subscription) equal(with *Subscription) bool {
	if s.ServiceName != with.ServiceName {
		return false
	}
	if s.ServiceId != with.ServiceId {
		return false
	}
	if s.Internal != with.Internal {
		return false
	}
	if s.EndpointUrl != with.EndpointUrl {
		return false
	}
	if s.ReportNewBlock != with.ReportNewBlock {
		return false
	}
	if s.ReportIncomingTx != with.ReportIncomingTx {
		return false
	}
	if s.ReportOutgoingTx != with.ReportOutgoingTx {
		return false
	}
	if s.GatherToMaster != with.GatherToMaster {
		return false
	}
	if s.ReportMainCoin != with.ReportMainCoin {
		return false
	}
	if len(s.MasterList) != len(with.MasterList) {
		return false
	}
	for i, v := range s.MasterList {
		if v != with.MasterList[i] {
			return false
		}
	}
	if len(s.ReportTokens) != len(with.ReportTokens) {
		return false
	}
	for k, v := range s.ReportTokens {
		if v != with.ReportTokens[k] {
			return false
		}
	}
	return true
}

// sendNotification sends an RPC notification to the subscriber's endpoint.
// For internal subscriptions, logs the notification instead.
func (s *Subscription) sendNotification(method string, message interface{}, debug bool) {
	if s.Internal || s.EndpointUrl == "" {
		log.Debug("Internal notification:", method)
		log.Dump(message)
		return
	}
	if s.rpc == nil {
		s.rpc = urpc.NewClient(
			urpc.WithHTTPRpc(s.EndpointUrl, nil),
		)
	}
	req := urpc.NewRequestWithObject(method, message)
	//log.Debug("Send service notification to", s.EndpointUrl)
	//log.Dump(message)
	response, err := s.rpc.Call(req)
	//_, _ = s.rpc.Call(req)
	if err != nil {
		if debug {
			log.Error("Can not send service notification:", err, response)
		}
	}
}

// Signer defines an interface for signing notification payloads.
type Signer interface {
	Sign(apiKey string)
}
