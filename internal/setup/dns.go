package setup

import (
	"fmt"
	"log"

	"github.com/miekg/dns"
)

func StartDNSServer(redirectIP string) *dns.Server {
	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true

		for _, q := range r.Question {
			if q.Qtype == dns.TypeA {
				rr, _ := dns.NewRR(fmt.Sprintf("%s 3600 IN A %s", q.Name, redirectIP))
				m.Answer = append(m.Answer, rr)
			}
		}
		w.WriteMsg(m)
	})

	server := &dns.Server{Addr: ":53", Net: "udp"}
	
	go func() {
		log.Printf("[DNS] Starting DNS Spoofing Server on :53 -> %s", redirectIP)
		if err := server.ListenAndServe(); err != nil {
			log.Printf("[DNS] Failed to start server: %v", err)
		}
	}()

	return server
}