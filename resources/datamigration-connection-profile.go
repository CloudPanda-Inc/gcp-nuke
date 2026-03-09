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

const DatabaseMigrationConnectionProfileResource = "DatabaseMigrationConnectionProfile"

func init() {
	registry.Register(&registry.Registration{
		Name:      DatabaseMigrationConnectionProfileResource,
		Scope:     nuke.Project,
		Resource:  &DatabaseMigrationConnectionProfile{},
		Lister:    &DatabaseMigrationConnectionProfileLister{},
		DependsOn: []string{DatabaseMigrationJobResource},
	})
}

type DatabaseMigrationConnectionProfileLister struct {
	svc *clouddms.DataMigrationClient
}

func (l *DatabaseMigrationConnectionProfileLister) Close() {
	if l.svc != nil {
		_ = l.svc.Close()
	}
}

func (l *DatabaseMigrationConnectionProfileLister) List(ctx context.Context, o interface{}) ([]resource.Resource, error) {
	var resources []resource.Resource

	opts := o.(*nuke.ListerOpts)
	if err := opts.BeforeList(nuke.Regional, "datamigration.googleapis.com", DatabaseMigrationConnectionProfileResource); err != nil {
		return resources, err
	}

	if l.svc == nil {
		var err error
		l.svc, err = clouddms.NewDataMigrationClient(ctx, opts.ClientOptions...)
		if err != nil {
			return nil, err
		}
	}

	req := &clouddmspb.ListConnectionProfilesRequest{
		Parent: "projects/" + *opts.Project + "/locations/" + *opts.Region,
	}

	it := l.svc.ListConnectionProfiles(ctx, req)
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
			logrus.WithError(err).Error("unable to iterate database migration connection profiles")
			break
		}

		nameParts := strings.Split(resp.Name, "/")
		name := nameParts[len(nameParts)-1]

		resources = append(resources, &DatabaseMigrationConnectionProfile{
			svc:         l.svc,
			Project:     opts.Project,
			Region:      opts.Region,
			FullName:    ptr.String(resp.Name),
			Name:        ptr.String(name),
			DisplayName: ptr.String(resp.DisplayName),
			State:       ptr.String(resp.State.String()),
			Provider:    ptr.String(resp.Provider.String()),
		})
	}

	return resources, nil
}

type DatabaseMigrationConnectionProfile struct {
	svc         *clouddms.DataMigrationClient
	Project     *string
	Region      *string
	FullName    *string
	Name        *string `description:"The name of the connection profile"`
	DisplayName *string `description:"The display name of the connection profile"`
	State       *string `description:"The current state of the connection profile"`
	Provider    *string `description:"The database provider (e.g. CLOUDSQL, RDS, AURORA)"`
}

func (r *DatabaseMigrationConnectionProfile) Remove(ctx context.Context) error {
	op, err := r.svc.DeleteConnectionProfile(ctx, &clouddmspb.DeleteConnectionProfileRequest{
		Name:  *r.FullName,
		Force: true,
	})
	if err != nil {
		return err
	}
	return op.Wait(ctx)
}

func (r *DatabaseMigrationConnectionProfile) Properties() types.Properties {
	return types.NewPropertiesFromStruct(r)
}

func (r *DatabaseMigrationConnectionProfile) String() string {
	return *r.Name
}
