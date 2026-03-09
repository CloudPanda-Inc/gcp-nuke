package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gotidy/ptr"
	"github.com/sirupsen/logrus"

	"google.golang.org/api/iterator"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"

	"github.com/ekristen/libnuke/pkg/registry"
	"github.com/ekristen/libnuke/pkg/resource"
	"github.com/ekristen/libnuke/pkg/types"

	"github.com/ekristen/gcp-nuke/pkg/nuke"
)

const CloudBuildTriggerResource = "CloudBuildTrigger"

func init() {
	registry.Register(&registry.Registration{
		Name:     CloudBuildTriggerResource,
		Scope:    nuke.Project,
		Resource: &CloudBuildTrigger{},
		Lister:   &CloudBuildTriggerLister{},
	})
}

type CloudBuildTriggerLister struct {
	svc *cloudbuild.Client
}

func (l *CloudBuildTriggerLister) Close() {
	if l.svc != nil {
		_ = l.svc.Close()
	}
}

func (l *CloudBuildTriggerLister) List(ctx context.Context, o interface{}) ([]resource.Resource, error) {
	var resources []resource.Resource

	opts := o.(*nuke.ListerOpts)
	if err := opts.BeforeList(nuke.Regional, "cloudbuild.googleapis.com", CloudBuildTriggerResource); err != nil {
		return resources, err
	}

	if l.svc == nil {
		var err error
		l.svc, err = cloudbuild.NewRESTClient(ctx, opts.ClientOptions...)
		if err != nil {
			return nil, err
		}
	}

	req := &cloudbuildpb.ListBuildTriggersRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", *opts.Project, *opts.Region),
	}

	it := l.svc.ListBuildTriggers(ctx, req)
	for {
		resp, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "not a valid location") ||
				strings.Contains(err.Error(), "is not enabled") {
				return resources, nil
			}
			logrus.WithError(err).Error("unable to iterate cloud build triggers")
			break
		}

		nameParts := strings.Split(resp.ResourceName, "/")
		name := nameParts[len(nameParts)-1]

		resources = append(resources, &CloudBuildTrigger{
			svc:       l.svc,
			Project:   opts.Project,
			Region:    opts.Region,
			FullName:  ptr.String(resp.ResourceName),
			Name:      ptr.String(resp.Name),
			TriggerID: ptr.String(name),
		})
	}

	return resources, nil
}

type CloudBuildTrigger struct {
	svc       *cloudbuild.Client
	Project   *string
	Region    *string
	FullName  *string
	Name      *string `description:"The user-defined name of the trigger"`
	TriggerID *string `description:"The unique identifier of the trigger"`
}

func (r *CloudBuildTrigger) Remove(ctx context.Context) error {
	return r.svc.DeleteBuildTrigger(ctx, &cloudbuildpb.DeleteBuildTriggerRequest{
		Name: *r.FullName,
	})
}

func (r *CloudBuildTrigger) Properties() types.Properties {
	return types.NewPropertiesFromStruct(r)
}

func (r *CloudBuildTrigger) String() string {
	return *r.Name
}
