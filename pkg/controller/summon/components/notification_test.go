/*
Copyright 2018-2019 Ridecell, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package components_test

import (
	"fmt"
	"strconv"

	"github.com/nlopes/slack"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("SummonPlatform Notification Component", func() {
	comp := summoncomponents.NewNotification()
	var mockedSlackClient *summoncomponents.SlackClientMock
	var mockedDeployStatusClient *summoncomponents.DeployStatusClientMock

	BeforeEach(func() {
		comp = summoncomponents.NewNotification()
		mockedSlackClient = &summoncomponents.SlackClientMock{
			PostMessageFunc: func(_ string, _ slack.Attachment) (string, string, error) {
				return "", "", nil
			},
		}
		comp.InjectSlackClient(mockedSlackClient)

		instance.Spec.Notifications.SlackChannel = "#test-channel"
		// Defaults component would do this, but unit tests doesn't run defaults component.
		instance.Spec.Environment = "dev"

		mockedDeployStatusClient = &summoncomponents.DeployStatusClientMock{
			PostStatusFunc: func(_, _, _, _ string) error {
				return nil
			},
		}
		comp.InjectDeployStatusClient(mockedDeployStatusClient)
	})

	Describe("WatchTypes", func() {
		It("has none", func() {
			types := comp.WatchTypes()
			Expect(types).To(BeEmpty())
		})
	})

	Describe("Reconcile", func() {
		It("does nothing if status is initializing", func() {
			instance.Status.Status = summonv1beta1.StatusInitializing
			Expect(comp).To(ReconcileContext(ctx))
			Expect(mockedSlackClient.PostMessageCalls()).To(HaveLen(0))
			Expect(mockedDeployStatusClient.PostStatusCalls()).To(HaveLen(0))
		})

		It("does nothing if status is migrating", func() {
			instance.Status.Status = summonv1beta1.StatusMigrating
			Expect(comp).To(ReconcileContext(ctx))
			Expect(mockedSlackClient.PostMessageCalls()).To(HaveLen(0))
			Expect(mockedDeployStatusClient.PostStatusCalls()).To(HaveLen(0))
		})

		It("does nothing if status is deploying", func() {
			instance.Status.Status = summonv1beta1.StatusDeploying
			Expect(comp).To(ReconcileContext(ctx))
			Expect(mockedSlackClient.PostMessageCalls()).To(HaveLen(0))
			Expect(mockedDeployStatusClient.PostStatusCalls()).To(HaveLen(0))
		})

		It("sends a success notification on a new deployment", func() {
			instance.Spec.Version = "1234-eb6b515-master"
			instance.Status.Notification.NotifyVersion = ""
			instance.Status.Status = summonv1beta1.StatusReady
			Expect(comp).To(ReconcileContext(ctx))
			Expect(mockedSlackClient.PostMessageCalls()).To(HaveLen(1))
			post := mockedSlackClient.PostMessageCalls()[0]
			Expect(post.In1).To(Equal("#test-channel"))
			Expect(post.In2.Title).To(Equal("foo.ridecell.us Deployment"))
			Expect(post.In2.Fallback).To(Equal("foo.ridecell.us deployed version 1234-eb6b515-master successfully"))
			Expect(post.In2.Fields[0].Value).To(Equal("<https://github.com/Ridecell/summon-platform/tree/eb6b515|eb6b515>"))
			Expect(instance.Status.Notification.NotifyVersion).To(Equal("1234-eb6b515-master"))
			Expect(mockedDeployStatusClient.PostStatusCalls()).To(HaveLen(1))
			deployPost := mockedDeployStatusClient.PostStatusCalls()[0]
			Expect(deployPost.Name).To(Equal("foo"))
			Expect(deployPost.Env).To(Equal("dev"))
			Expect(deployPost.Tag).To(Equal("1234-eb6b515-master"))
		})

		It("sends a success notification on a new deployment to additional slack channels", func() {
			instance.Spec.Notifications.SlackChannels = []string{"#test-channel-2"}
			instance.Spec.Version = "1234-eb6b515-master"
			instance.Status.Notification.NotifyVersion = ""
			instance.Status.Status = summonv1beta1.StatusReady
			Expect(comp).To(ReconcileContext(ctx))
			Expect(mockedSlackClient.PostMessageCalls()).To(HaveLen(2))
			post := mockedSlackClient.PostMessageCalls()[0]
			Expect(post.In1).To(Equal("#test-channel"))
			Expect(post.In2.Title).To(Equal("foo.ridecell.us Deployment"))
			Expect(post.In2.Fallback).To(Equal("foo.ridecell.us deployed version 1234-eb6b515-master successfully"))
			Expect(post.In2.Fields[0].Value).To(Equal("<https://github.com/Ridecell/summon-platform/tree/eb6b515|eb6b515>"))
			post2 := mockedSlackClient.PostMessageCalls()[1]
			Expect(post2.In1).To(Equal("#test-channel-2"))
			Expect(post2.In2.Title).To(Equal("foo.ridecell.us Deployment"))
			Expect(post2.In2.Fallback).To(Equal("foo.ridecell.us deployed version 1234-eb6b515-master successfully"))
			Expect(post2.In2.Fields[0].Value).To(Equal("<https://github.com/Ridecell/summon-platform/tree/eb6b515|eb6b515>"))
			Expect(instance.Status.Notification.NotifyVersion).To(Equal("1234-eb6b515-master"))
			Expect(mockedDeployStatusClient.PostStatusCalls()).To(HaveLen(1))
		})

		It("does not send a success notification on an existing deployment", func() {
			instance.Spec.Notifications.SlackChannels = []string{"#test-channel-2", "#test-channel-3"}
			instance.Spec.Version = "1234-eb6b515-master"
			instance.Status.Notification.NotifyVersion = "1234-eb6b515-master"
			instance.Status.Status = summonv1beta1.StatusReady
			Expect(comp).To(ReconcileContext(ctx))
			Expect(mockedSlackClient.PostMessageCalls()).To(HaveLen(0))
			Expect(instance.Status.Notification.NotifyVersion).To(Equal("1234-eb6b515-master"))
			Expect(mockedDeployStatusClient.PostStatusCalls()).To(HaveLen(0))
		})

		It("does not set fields on a non-standard version", func() {
			// More importantly, it doesn't choke.
			instance.Spec.Version = "1234"
			instance.Status.Notification.NotifyVersion = ""
			instance.Status.Status = summonv1beta1.StatusReady
			Expect(comp).To(ReconcileContext(ctx))
			Expect(mockedSlackClient.PostMessageCalls()).To(HaveLen(1))
			post := mockedSlackClient.PostMessageCalls()[0]
			Expect(post.In2.Fallback).To(Equal("foo.ridecell.us deployed version 1234 successfully"))
			Expect(post.In2.Fields).To(HaveLen(0))
			Expect(instance.Status.Notification.NotifyVersion).To(Equal("1234"))
			Expect(mockedDeployStatusClient.PostStatusCalls()).To(HaveLen(1))
			deployPost := mockedDeployStatusClient.PostStatusCalls()[0]
			Expect(deployPost.Name).To(Equal("foo"))
			Expect(deployPost.Env).To(Equal("dev"))
			Expect(deployPost.Tag).To(Equal("1234"))
		})

		It("sends an error notification on a new error", func() {
			instance.Spec.Notifications.SlackChannels = []string{"#test-channel-2", "#test-channel-3"}
			instance.Status.Message = "Someone set us up the bomb"
			instance.Status.Status = summonv1beta1.StatusError
			Expect(comp).To(ReconcileContext(ctx))
			Expect(mockedSlackClient.PostMessageCalls()).To(HaveLen(3))
			for index, post := range mockedSlackClient.PostMessageCalls() {
				if index == 0 {
					Expect(post.In1).To(Equal("#test-channel"))
				} else {
					Expect(post.In1).To(Equal("#test-channel-" + strconv.Itoa(index+1)))
				}
				Expect(post.In2.Title).To(Equal("foo.ridecell.us Deployment"))
				Expect(post.In2.Fallback).To(Equal("foo.ridecell.us has error: Someone set us up the bomb"))
			}
			Expect(mockedDeployStatusClient.PostStatusCalls()).To(HaveLen(0))
		})

		It("does not send an error the second time for the same error", func() {
			instance.Status.Message = "Someone set us up the bomb"
			instance.Status.Status = summonv1beta1.StatusError
			Expect(comp).To(ReconcileContext(ctx))
			Expect(comp).To(ReconcileContext(ctx))
			Expect(mockedSlackClient.PostMessageCalls()).To(HaveLen(1))
			Expect(mockedDeployStatusClient.PostStatusCalls()).To(HaveLen(0))
		})

		It("sends two error notifications for two different errors", func() {
			instance.Status.Message = "Someone set us up the bomb"
			instance.Status.Status = summonv1beta1.StatusError
			Expect(comp).To(ReconcileContext(ctx))
			instance.Status.Message = "You have no chance to survive"
			Expect(comp).To(ReconcileContext(ctx))
			Expect(mockedSlackClient.PostMessageCalls()).To(HaveLen(2))
			post := mockedSlackClient.PostMessageCalls()[0]
			Expect(post.In1).To(Equal("#test-channel"))
			Expect(post.In2.Title).To(Equal("foo.ridecell.us Deployment"))
			Expect(post.In2.Fallback).To(Equal("foo.ridecell.us has error: Someone set us up the bomb"))
			post2 := mockedSlackClient.PostMessageCalls()[1]
			Expect(post2.In1).To(Equal("#test-channel"))
			Expect(post2.In2.Title).To(Equal("foo.ridecell.us Deployment"))
			Expect(post2.In2.Fallback).To(Equal("foo.ridecell.us has error: You have no chance to survive"))
			Expect(mockedDeployStatusClient.PostStatusCalls()).To(HaveLen(0))
		})

		It("sends two error notifications for two identical errors on different versions", func() {
			instance.Spec.Version = "v1"
			instance.Status.Message = "I thought what i'd do was i'd pretend"
			instance.Status.Status = summonv1beta1.StatusError
			Expect(comp).To(ReconcileContext(ctx))
			instance.Spec.Version = "v2"
			instance.Status.Message = "I thought what i'd do was i'd pretend"
			Expect(comp).To(ReconcileContext(ctx))
			Expect(mockedSlackClient.PostMessageCalls()).To(HaveLen(2))
			post := mockedSlackClient.PostMessageCalls()[0]
			Expect(post.In1).To(Equal("#test-channel"))
			Expect(post.In2.Title).To(Equal("foo.ridecell.us Deployment"))
			Expect(post.In2.Fallback).To(Equal("foo.ridecell.us has error: I thought what i'd do was i'd pretend"))
			post2 := mockedSlackClient.PostMessageCalls()[1]
			Expect(post2.In1).To(Equal("#test-channel"))
			Expect(post2.In2.Title).To(Equal("foo.ridecell.us Deployment"))
			Expect(post2.In2.Fallback).To(Equal("foo.ridecell.us has error: I thought what i'd do was i'd pretend"))
			Expect(mockedDeployStatusClient.PostStatusCalls()).To(HaveLen(0))
		})
	})

	Describe("ReconcileError", func() {
		It("sends an error notification on a new error", func() {
			Expect(comp).To(ReconcileErrorContext(ctx, fmt.Errorf("Someone set us up the bomb")))
			Expect(mockedSlackClient.PostMessageCalls()).To(HaveLen(1))
			post := mockedSlackClient.PostMessageCalls()[0]
			Expect(post.In1).To(Equal("#test-channel"))
			Expect(post.In2.Title).To(Equal("foo.ridecell.us Deployment"))
			Expect(post.In2.Fallback).To(Equal("foo.ridecell.us has error: Someone set us up the bomb"))
			Expect(mockedDeployStatusClient.PostStatusCalls()).To(HaveLen(0))
		})

		It("does not send an error the second time for the same error", func() {
			instance.Spec.Notifications.SlackChannels = []string{"#tchannel-2", "#tchannel-3"}
			Expect(comp).To(ReconcileErrorContext(ctx, fmt.Errorf("Someone set us up the bomb")))
			Expect(comp).To(ReconcileErrorContext(ctx, fmt.Errorf("Someone set us up the bomb")))
			Expect(mockedSlackClient.PostMessageCalls()).To(HaveLen(3))
			Expect(mockedDeployStatusClient.PostStatusCalls()).To(HaveLen(0))
		})

		It("sends two error notifications for two different errors (to both primary and additional slack channels)", func() {
			instance.Spec.Notifications.SlackChannels = []string{"#otherchannel"}
			Expect(comp).To(ReconcileErrorContext(ctx, fmt.Errorf("Someone set us up the bomb")))
			Expect(comp).To(ReconcileErrorContext(ctx, fmt.Errorf("You have no chance to survive")))
			// 2 errors to primary channel, 2 errors to additional channel
			Expect(mockedSlackClient.PostMessageCalls()).To(HaveLen(4))
			post := mockedSlackClient.PostMessageCalls()[0]
			Expect(post.In1).To(Equal("#test-channel"))
			Expect(post.In2.Title).To(Equal("foo.ridecell.us Deployment"))
			Expect(post.In2.Fallback).To(Equal("foo.ridecell.us has error: Someone set us up the bomb"))
			post2 := mockedSlackClient.PostMessageCalls()[1]
			Expect(post2.In1).To(Equal("#otherchannel"))
			Expect(post2.In2.Title).To(Equal("foo.ridecell.us Deployment"))
			Expect(post2.In2.Fallback).To(Equal("foo.ridecell.us has error: Someone set us up the bomb"))
			post3 := mockedSlackClient.PostMessageCalls()[2]
			Expect(post3.In1).To(Equal("#test-channel"))
			Expect(post3.In2.Title).To(Equal("foo.ridecell.us Deployment"))
			Expect(post3.In2.Fallback).To(Equal("foo.ridecell.us has error: You have no chance to survive"))
			post4 := mockedSlackClient.PostMessageCalls()[3]
			Expect(post4.In1).To(Equal("#otherchannel"))
			Expect(post4.In2.Title).To(Equal("foo.ridecell.us Deployment"))
			Expect(post4.In2.Fallback).To(Equal("foo.ridecell.us has error: You have no chance to survive"))
			Expect(mockedDeployStatusClient.PostStatusCalls()).To(HaveLen(0))
		})
	})
})
