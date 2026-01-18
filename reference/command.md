## Create Migration
migrate create -ext sql -dir db/migrations -seq create_users_table


# buat migration
make migrate create name=create_users_table

# jalankan migration
make migrate-up

# generate sqlc
make sqlc

# run test
make test

# run app
make run
