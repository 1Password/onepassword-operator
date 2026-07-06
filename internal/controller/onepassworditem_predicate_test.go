package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	onepasswordv1 "github.com/1Password/onepassword-operator/api/v1"
)

var _ = Describe("OnePasswordItem label selector predicate", func() {
	selector := metav1.LabelSelector{
		MatchLabels: map[string]string{"operator.1password.io/owner": "vouchercodes"},
	}

	newItem := func(labels map[string]string) *onepasswordv1.OnePasswordItem {
		return &onepasswordv1.OnePasswordItem{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "item",
				Namespace: "default",
				Labels:    labels,
			},
		}
	}

	It("reconciles items whose labels match the selector", func() {
		pred, err := predicate.LabelSelectorPredicate(selector)
		Expect(err).ToNot(HaveOccurred())

		item := newItem(map[string]string{"operator.1password.io/owner": "vouchercodes"})
		Expect(pred.Create(event.CreateEvent{Object: item})).To(BeTrue())
		Expect(pred.Update(event.UpdateEvent{ObjectOld: item, ObjectNew: item})).To(BeTrue())
		Expect(pred.Delete(event.DeleteEvent{Object: item})).To(BeTrue())
		Expect(pred.Generic(event.GenericEvent{Object: item})).To(BeTrue())
	})

	It("ignores items whose labels do not match the selector", func() {
		pred, err := predicate.LabelSelectorPredicate(selector)
		Expect(err).ToNot(HaveOccurred())

		other := newItem(map[string]string{"operator.1password.io/owner": "vc-services"})
		Expect(pred.Create(event.CreateEvent{Object: other})).To(BeFalse())
		Expect(pred.Update(event.UpdateEvent{ObjectOld: other, ObjectNew: other})).To(BeFalse())
		Expect(pred.Delete(event.DeleteEvent{Object: other})).To(BeFalse())
		Expect(pred.Generic(event.GenericEvent{Object: other})).To(BeFalse())

		unlabeled := newItem(nil)
		Expect(pred.Create(event.CreateEvent{Object: unlabeled})).To(BeFalse())
	})

	It("returns an error for an invalid selector", func() {
		_, err := metav1.ParseToLabelSelector("=bad=selector=")
		Expect(err).To(HaveOccurred())
	})
})
