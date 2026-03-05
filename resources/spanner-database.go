package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gotidy/ptr"
	"github.com/sirupsen/logrus"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"google.golang.org/api/iterator"

	"github.com/ekristen/libnuke/pkg/registry"
	"github.com/ekristen/libnuke/pkg/resource"
	"github.com/ekristen/libnuke/pkg/settings"
	"github.com/ekristen/libnuke/pkg/types"

	"github.com/ekristen/gcp-nuke/pkg/nuke"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

const SpannerDatabaseResource = "SpannerDatabase"

func init() {
	registry.Register(&registry.Registration{
		Name:     SpannerDatabaseResource,
		Scope:    nuke.Project,
		Resource: &SpannerDatabase{},
		Lister:   &SpannerDatabaseLister{},
		Settings: []string{
			"DisableDeletionProtection",
		},
	})
}

type SpannerDatabaseLister struct {
	svc          *database.DatabaseAdminClient
	instancesSvc *instance.InstanceAdminClient
}

func (l *SpannerDatabaseLister) Close() {
	if l.svc != nil {
		_ = l.svc.Close()
	}
	if l.instancesSvc != nil {
		_ = l.instancesSvc.Close()
	}
}

func (l *SpannerDatabaseLister) List(ctx context.Context, o interface{}) ([]resource.Resource, error) {
	var resources []resource.Resource

	opts := o.(*nuke.ListerOpts)
	if err := opts.BeforeList(nuke.Global, "spanner.googleapis.com", SpannerDatabaseResource); err != nil {
		return resources, err
	}

	if l.svc == nil {
		var err error
		l.svc, err = database.NewDatabaseAdminClient(ctx, opts.ClientOptions...)
		if err != nil {
			return nil, err
		}
	}

	if l.instancesSvc == nil {
		var err error
		l.instancesSvc, err = instance.NewInstanceAdminClient(ctx, opts.ClientOptions...)
		if err != nil {
			return nil, err
		}
	}

	instanceReq := &instancepb.ListInstancesRequest{
		Parent: "projects/" + *opts.Project,
	}

	instanceIt := l.instancesSvc.ListInstances(ctx, instanceReq)
	for {
		inst, err := instanceIt.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			logrus.WithError(err).Error("unable to iterate spanner instances")
			break
		}

		dbReq := &databasepb.ListDatabasesRequest{
			Parent: inst.Name,
		}

		dbIt := l.svc.ListDatabases(ctx, dbReq)
		for {
			db, err := dbIt.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				logrus.WithError(err).Error("unable to iterate spanner databases")
				break
			}

			nameParts := strings.Split(db.Name, "/")
			name := nameParts[len(nameParts)-1]

			instanceParts := strings.Split(inst.Name, "/")
			instanceName := instanceParts[len(instanceParts)-1]

			resources = append(resources, &SpannerDatabase{
				svc:      l.svc,
				Project:  opts.Project,
				FullName: ptr.String(db.Name),
				Name:     ptr.String(name),
				Instance: ptr.String(instanceName),
				State:    ptr.String(db.State.String()),
			})
		}
	}

	return resources, nil
}

type SpannerDatabase struct {
	svc      *database.DatabaseAdminClient
	settings *settings.Setting
	Project  *string
	FullName *string
	Name     *string `description:"The name of the Spanner database"`
	Instance *string `description:"The instance this database belongs to"`
	State    *string `description:"The current state of the database"`
}

func (r *SpannerDatabase) Remove(ctx context.Context) error {
	if err := r.disableDeletionProtection(ctx); err != nil {
		return err
	}

	return r.svc.DropDatabase(ctx, &databasepb.DropDatabaseRequest{
		Database: *r.FullName,
	})
}

func (r *SpannerDatabase) Settings(setting *settings.Setting) {
	r.settings = setting
}

func (r *SpannerDatabase) disableDeletionProtection(ctx context.Context) error {
	if r.settings == nil || !r.settings.GetBool("DisableDeletionProtection") {
		return nil
	}

	op, err := r.svc.UpdateDatabase(ctx, &databasepb.UpdateDatabaseRequest{
		Database: &databasepb.Database{
			Name:                 *r.FullName,
			EnableDropProtection: false,
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"enable_drop_protection"},
		},
	})
	if err != nil {
		return fmt.Errorf("unable to disable deletion protection: %w", err)
	}

	if _, err = op.Wait(ctx); err != nil {
		return fmt.Errorf("unable to wait for deletion protection update operation: %w", err)
	}

	return nil
}

func (r *SpannerDatabase) Properties() types.Properties {
	return types.NewPropertiesFromStruct(r)
}

func (r *SpannerDatabase) String() string {
	return *r.Name
}
