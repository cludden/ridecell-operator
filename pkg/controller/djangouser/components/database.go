/*
Copyright 2018 Ridecell, Inc..

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

package components

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/dbpool"
)

type databaseComponent struct{}

func NewDatabase() *databaseComponent {
	return &databaseComponent{}
}

func (_ *databaseComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *databaseComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *databaseComponent) Reconcile(ctx *components.ComponentContext) (reconcile.Result, error) {
	instance := ctx.Top.(*summonv1beta1.DjangoUser)

	// Try to find the password to use.
	secret := &corev1.Secret{}
	err := ctx.Get(ctx.Context, types.NamespacedName{Name: instance.Spec.PasswordSecret, Namespace: instance.Namespace}, secret)
	if err != nil {
		return reconcile.Result{Requeue: true}, fmt.Errorf("database: Unable to load password secret: %v", err)
	}
	password, ok := secret.Data["password"]
	if !ok {
		return reconcile.Result{Requeue: true}, fmt.Errorf("database: Password secret has no key \"password\"")
	}
	hashedPassword, err := comp.hashPassword(password)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("database: Error hashing password: %v", err)
	}

	// Connect to the database.
	db, err := comp.openDatabase(ctx)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	// Big ass SQL.
	query := `
INSERT INTO auth_user (username, password, first_name, last_name, email, is_active, is_staff, is_superuser, date_joined)
  VALUES ($1, $2, $3, $4, $1, $5, $6, $7, NOW())
  ON CONFLICT (username) DO UPDATE SET
    password=EXCLUDED.password,
    first_name=EXCLUDED.first_name,
    last_name=EXCLUDED.last_name,
    email=EXCLUDED.email,
    is_active=EXCLUDED.is_active,
    is_staff=EXCLUDED.is_staff,
    is_superuser=EXCLUDED.is_superuser
  RETURNING id;`

	// Run the query.
	row := db.QueryRow(query, instance.Spec.Email, hashedPassword, instance.Spec.FirstName, instance.Spec.LastName, instance.Spec.Active, instance.Spec.Staff, instance.Spec.Superuser)
	var id int
	err = row.Scan(&id)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("database: Error running query: %v", err)
	}

	// Success!
	instance.Status.Status = summonv1beta1.StatusReady
	instance.Status.Message = fmt.Sprintf("User %v created", id)

	return reconcile.Result{}, nil
}

func (comp *databaseComponent) hashPassword(password []byte) (string, error) {
	// Take the SHA256.
	digested := sha256.Sum256(password)

	// Hex encode it.
	encoded := make([]byte, hex.EncodedLen(len(digested)))
	hex.Encode(encoded, digested[:])

	// Bcrypt it.
	hashed, err := bcrypt.GenerateFromPassword(encoded, 12)
	if err != nil {
		return "", err
	}

	// Format like Django uses.
	return fmt.Sprintf("bcrypt_sha256$%s", hashed), nil
}

func (comp *databaseComponent) openDatabase(ctx *components.ComponentContext) (*sql.DB, error) {
	instance := ctx.Top.(*summonv1beta1.DjangoUser)
	dbInfo := instance.Spec.Database
	passwordSecret := &corev1.Secret{}
	err := ctx.Get(ctx.Context, types.NamespacedName{Name: dbInfo.PasswordSecretRef.Name, Namespace: instance.Namespace}, passwordSecret)
	if err != nil {
		return nil, fmt.Errorf("database: Unable to load database secret %v: %v", dbInfo.PasswordSecretRef.Name, err)
	}
	dbPassword, ok := passwordSecret.Data[dbInfo.PasswordSecretRef.Key]
	if !ok {
		return nil, fmt.Errorf("database: Password key %v not found in database secret %v", dbInfo.PasswordSecretRef.Key, dbInfo.PasswordSecretRef.Name)
	}
	connStr := fmt.Sprintf("host=%s port=%v dbname=%s user=%v password='%s' sslmode=verify-full", dbInfo.Host, dbInfo.Port, dbInfo.Database, dbInfo.Username, dbPassword)
	db, err := dbpool.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("database: Unable to open database connection: %v", err)
	}
	return db, nil
}