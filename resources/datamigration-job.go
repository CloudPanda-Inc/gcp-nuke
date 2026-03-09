package resources

import (
	"context"
	"errors"
	"strings"

	"github.com/gotidy/ptr"
	"github.com/sirupsen/logrus"

	"google.golang.org/api/iterator"

	clouddms "cloud.google.com/go/clouddms/apiv1"
	"cloud.google.com/go/clouddms/apiv1/clouddmspb"

	"github.com/ekristen/libnuke/pkg/registry"
	"github.com/ekristen/libnuke/pkg/resource"
	"github.com/ekristen/libnuke/pkg/types"

	"github.com/ekristen/gcp-nuke/pkg/nuke"
)

const DatabaseMigrationJobResource = "DatabaseMigrationJob"

func init() {
	registry.Register(&registry.Registration{
		Name:     DatabaseMigrationJobResource,
		Scope:    nuke.Project,
		Resource: &DatabaseMigrationJob{},
		Lister:   &DatabaseMigrationJobLister{},
	})
}

type DatabaseMigrationJobLister struct {
	svc *clouddms.DataMigrationClient
}

func (l *DatabaseMigrationJobLister) Close() {
	if l.svc != nil {
		_ = l.svc.Close()
	}
}

func (l *DatabaseMigrationJobLister) List(ctx context.Context, o interface{}) ([]resource.Resource, error) {
	var resources []resource.Resource

	opts := o.(*nuke.ListerOpts)
	if err := opts.BeforeList(nuke.Regional, "datamigration.googleapis.com", DatabaseMigrationJobResource); err != nil {
		return resources, err
	}

	if l.svc == nil {
		var err error
		l.svc, err = clouddms.NewDataMigrationClient(ctx, opts.ClientOptions...)
		if err != nil {
			return nil, err
		}
	}

	req := &clouddmspb.ListMigrationJobsRequest{
		Parent: "projects/" + *opts.Project + "/locations/" + *opts.Region,
	}

	it := l.svc.ListMigrationJobs(ctx, req)
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
			logrus.WithError(err).Error("unable to iterate database migration jobs")
			break
		}

		nameParts := strings.Split(resp.Name, "/")
		name := nameParts[len(nameParts)-1]

		resources = append(resources, &DatabaseMigrationJob{
			svc:      l.svc,
			Project:  opts.Project,
			Region:   opts.Region,
			FullName: ptr.String(resp.Name),
			Name:     ptr.String(name),
			State:    ptr.String(resp.State.String()),
			Type:     ptr.String(resp.Type.String()),
		})
	}

	return resources, nil
}

type DatabaseMigrationJob struct {
	svc      *clouddms.DataMigrationClient
	Project  *string
	Region   *string
	FullName *string
	Name     *string `description:"The name of the migration job"`
	State    *string `description:"The current state of the migration job"`
	Type     *string `description:"The type of the migration job (ONE_TIME or CONTINUOUS)"`
}

func (r *DatabaseMigrationJob) Remove(ctx context.Context) error {
	op, err := r.svc.DeleteMigrationJob(ctx, &clouddmspb.DeleteMigrationJobRequest{
		Name:  *r.FullName,
		Force: true,
	})
	if err != nil {
		return err
	}
	return op.Wait(ctx)
}

func (r *DatabaseMigrationJob) Properties() types.Properties {
	return types.NewPropertiesFromStruct(r)
}

func (r *DatabaseMigrationJob) String() string {
	return *r.Name
}
