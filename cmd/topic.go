// Copyright © 2018 Microsoft Corporation and contributors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/marstr/guid"

	"github.com/Azure/azure-sdk-for-go/services/eventgrid/2018-01-01/eventgrid"
	egmgmt "github.com/Azure/azure-sdk-for-go/services/eventgrid/mgmt/2018-01-01/eventgrid"
	"github.com/Azure/eventgrid-emulator/model"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const helpPageURI = "https://github.com/Azure/eventgrid-emulator"

var topicConfig = viper.New()

// topicCmd represents the topic command
var topicCmd = &cobra.Command{
	Use:   "topic",
	Short: "Starts a web server to respond to ",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		if crd := topicConfig.GetString("callback-retry-duration"); crd != "" {
			val, err := time.ParseDuration(crd)
			if err != nil {
				fmt.Fprintf(os.Stderr, "unable to parse %q as a duration. See https://godoc.org/time#ParseDuration for mor information.\n", crd)
				os.Exit(1)
			}
			model.SetCallbackRetryDuration(val)
		}

		fmt.Printf("Starting Event Grid Topic Emulator on port %d\n", topicConfig.GetInt("port"))
		fmt.Println(http.ListenAndServe(fmt.Sprintf(":%d", topicConfig.GetInt("port")), nil))
	},
}

func init() {
	rootCmd.AddCommand(topicCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// topicCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// topicCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	topicCmd.Flags().IntP("port", "p", 80, "The port that should be used to host the server emulating an EventGrid Topic.")
	topicConfig.BindPFlag("port", topicCmd.Flags().Lookup("port"))

	topicCmd.Flags().String("callback-retry-duration", "24h", "A time duration string as specified by ")
	topicConfig.BindPFlag("callback-retry-duration", topicCmd.Flags().Lookup("callback-retry-duration"))

	http.HandleFunc("/api/events", ProcessEventsHandler)
	http.HandleFunc("/subscribe", RegisterSubscriberHandler)
	http.HandleFunc("/", RedirectToHelpPage)
}

// RedirectToHelpPage instructs a client to retry the request, but instead to the GitHub page for
// this product.
func RedirectToHelpPage(resp http.ResponseWriter, req *http.Request) {
	requestID := guid.NewGUID()
	log.Printf("%v : redirecting to help page", requestID)
	resp.Header().Add("Location", helpPageURI)
	resp.WriteHeader(http.StatusTemporaryRedirect)
	return
}

// ProcessEventsHandler reads an HTTP Request that informs an Event Grid topic of an Event.
// It then relays that message to all subscribers who have not filtered out messages
// of this type and subject.
func ProcessEventsHandler(resp http.ResponseWriter, req *http.Request) {
	if !strings.EqualFold(req.Method, http.MethodPost) {
		fmt.Fprint(resp, "only POST is supported.")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	requestID := guid.NewGUID()

	log.Printf("%v : new event received", requestID)
	const MaxPayloadSize = 1024 * 1024
	const MaxEventSize = 64 * 1024

	limitedBody := io.LimitReader(req.Body, MaxPayloadSize)

	var payload eventgrid.Event

	if contents, err := ioutil.ReadAll(limitedBody); err == nil {
		if err = json.Unmarshal(contents, &payload); err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(resp, err)
			log.Printf("%v : unable to unmarshal request, %v", requestID, err)
			return
		}
	} else {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(resp, "unable to parse request body:", err)
		log.Printf("%v : unable to parse request body, %v", requestID, err)
		return
	}

	go log.Printf("%v : successfully processed event", requestID)
}

// RegisterSubscriberHandler mimics the ARM behavior of adding a subscriber to an Event Grid Topic.
//
// Note: It is important to understand that in a production app, this functionality wouldn't be used
// as the container hosting the webserver intialized. Rather, the list of subscribers would be created
// using Azure Resource Management (ARM) at the same time the Topic is created. You may also add
// listeners as new responses to events are needed.
func RegisterSubscriberHandler(resp http.ResponseWriter, req *http.Request) {
	const MaxPayloadSize = 1024 * 1024

	var payload egmgmt.EventSubscription

	limitedBody := io.LimitReader(req.Body, MaxPayloadSize)

	if contents, err := ioutil.ReadAll(limitedBody); err == nil {
		err = json.Unmarshal(contents, &payload)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(resp, err)
			return
		}
	} else {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(resp, err)
	}

	// filter := egmgmt.EventSubscriptionFilter{
	// 	IncludedEventTypes: &[]string{"all"},
	// }

	// if nil != payload.EventSubscriptionProperties.Filter {
	// 	filter = *payload.EventSubscriptionProperties.Filter
	// }

	//model.Register(*payload.EventSubscriptionProperties.Destination.AsWebHookEventSubscriptionDestination().EndpointURL, filter)
	resp.WriteHeader(http.StatusOK)
	return
}
