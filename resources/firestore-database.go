package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/gotidy/ptr"

	admin "cloud.google.com/go/firestore/apiv1/admin"
	"cloud.google.com/go/firestore/apiv1/admin/adminpb"

	"github.com/ekristen/libnuke/pkg/registry"
	"github.com/ekristen/libnuke/pkg/resource"
	"github.com/ekristen/libnuke/pkg/settings"
	"github.com/ekristen/libnuke/pkg/types"

	"github.com/ekristen/gcp-nuke/pkg/nuke"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

const FirestoreDatabaseResource = "FirestoreDatabase"

func init() {
	registry.Register(&registry.Registration{
		Name:     FirestoreDatabaseResource,
		Scope:    nuke.Project,
		Resource: &FirestoreDatabase{},
		Lister:   &FirestoreDatabaseLister{},
		Settings: []string{
			"DisableDeletionProtection",
		},
	})
}

type FirestoreDatabaseLister struct {
	svc *admin.FirestoreAdminClient
}

func (l *FirestoreDatabaseLister) Close() {
	if l.svc != nil {
		_ = l.svc.Close()
	}
}

func (l *FirestoreDatabaseLister) List(ctx context.Context, o interface{}) ([]resource.Resource, error) {
	opts := o.(*nuke.ListerOpts)
	var resources []resource.Resource
	if err := opts.BeforeList(nuke.Global, "firestore.googleapis.com", FirestoreDatabaseResource); err != nil {
		return resources, err
	}

	if l.svc == nil {
		var err error
		l.svc, err = admin.NewFirestoreAdminClient(ctx, opts.ClientOptions...)
		if err != nil {
			return nil, err
		}
	}

	req := &adminpb.ListDatabasesRequest{
		Parent: fmt.Sprintf("projects/%s", *opts.Project),
	}

	resp, err := l.svc.ListDatabases(ctx, req)
	if err != nil {
		return nil, err
	}

	for _, db := range resp.Databases {
		nameParts := strings.Split(db.Name, "/")

		resources = append(resources, &FirestoreDatabase{
			svc:      l.svc,
			project:  opts.Project,
			fullName: ptr.String(db.Name),
			Name:     ptr.String(nameParts[len(nameParts)-1]),
			Location: ptr.String(db.LocationId),
		})
	}

	return resources, nil
}

type FirestoreDatabase struct {
	svc      *admin.FirestoreAdminClient
	settings *settings.Setting
	project  *string
	fullName *string
	Name     *string
	Location *string
}

func (r *FirestoreDatabase) Remove(ctx context.Context) error {
	if err := r.disableDeletionProtection(ctx); err != nil {
		return err
	}

	_, err := r.svc.DeleteDatabase(ctx, &adminpb.DeleteDatabaseRequest{
		Name: *r.fullName,
	})
	return err
}

func (r *FirestoreDatabase) Settings(setting *settings.Setting) {
	r.settings = setting
}

func (r *FirestoreDatabase) disableDeletionProtection(ctx context.Context) error {
	if r.settings == nil || !r.settings.GetBool("DisableDeletionProtection") {
		return nil
	}

	op, err := r.svc.UpdateDatabase(ctx, &adminpb.UpdateDatabaseRequest{
		Database: &adminpb.Database{
			Name:                  *r.fullName,
			DeleteProtectionState: adminpb.Database_DELETE_PROTECTION_DISABLED,
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"delete_protection_state"},
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

func (r *FirestoreDatabase) Properties() types.Properties {
	return types.NewPropertiesFromStruct(r)
}

func (r *FirestoreDatabase) String() string {
	return *r.Name
}
