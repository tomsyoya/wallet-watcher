package store


import (
"context"
"errors"
"github.com/jackc/pgx/v5/pgxpool"
)


type Registration struct {
Chain string
Address string
}


type Store struct {
DB *pgxpool.Pool
}


func (s *Store) UpsertRegistration(ctx context.Context, r Registration) error {
if s.DB == nil {
return errors.New("db not initialized")
}
_, err := s.DB.Exec(ctx, `
INSERT INTO registrations (chain, address)
VALUES ($1, $2)
ON CONFLICT (chain, address) DO NOTHING;
`, r.Chain, r.Address)
return err
}

