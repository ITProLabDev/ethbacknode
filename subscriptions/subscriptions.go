package subscriptions

import (
	"backnode/clients/urpc"
	"backnode/tools/log"
	"encoding/json"
)

type ServiceId int

func NewSubscription(serviceId ServiceId, endpointUrl string, fillSettings func(s *Subscription)) *Subscription {
	subscription := &Subscription{
		ServiceId:   serviceId,
		EndpointUrl: endpointUrl,
	}
	fillSettings(subscription)
	return subscription
}

func (s *Manager) subscriptionsSave() error {
	b, err := json.MarshalIndent(s.subscribers, "", "\t")
	if err != nil {
		return err
	}
	return s.subscribersStorage.Save(b)
}
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

func (s *Manager) subscriptionsColdStart() error {
	s.subscribers[0] = &Subscription{
		ServiceId: 0,
		Internal:  true,
	}
	b, _ := json.MarshalIndent(s.subscribers, "", "\t")
	return s.subscribersStorage.Save(b)
}

func (s *Manager) subscriptionViewAll(viewver func(s *Subscription)) {
	s.subscribersMux.RLock()
	defer s.subscribersMux.RUnlock()
	for _, v := range s.subscribers {
		viewver(v)
	}
}

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

type Subscription struct {
	rpc                 *urpc.Client
	ServiceName         string          `json:"serviceName"`
	ServiceId           ServiceId       `json:"serviceId"`
	Internal            bool            `json:"internal,omitempty"`
	ApiToken            string          `json:"apiToken"`
	ApiKey              string          `json:"apiKey"`
	EndpointUrl         string          `json:"eventUrl"`
	ReportNewBlock      bool            `json:"reportNewBlock"`
	ReportIncomingTx    bool            `json:"reportIncomingTx"`
	ReportOutgoingTx    bool            `json:"reportOutgoingTx"`
	ReportMainCoin      bool            `json:"reportMainCoin"`
	ReportTokens        map[string]bool `json:"reportTokens"`
	ReportBalanceChange bool            `json:"balanceChange"`
	GatherToMaster      bool            `json:"gatherToMaster"`
	MasterList          []string        `json:"masterList"`
}

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

type Signer interface {
	Sign(apiKey string)
}
