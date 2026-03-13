# TODO
This file contains feature or bugfix requests

## Add `Administrator` attribute to User
Add a boolen field to the User database table and struct to indicate whether
this user is an admin or not.

Return this new field in the `login` response along with the token, so a client
knows whether the authenticated user is an admin or not.

Attach the user id to the JWT token (perhaps as part of the "claims") generated
at `login` to indicate that the token is valid for an admin.

Split up the privileged router into `privileged` and `admin`, leaving routes
that use a GET method under `privileged` and moving those that use POST, PUT,
or DELETE into `admin`

Create a new `adminRequired` function to check that the user associated with
the passed token is an admin to restrict access to routes in the `admin` router

Add the field to `bootstrapping/users.csv`, setting user `foo` as an admin and
leaving the rest as non-administrative users.

Generate a MySQL query for altering the production users table to include the
`administrator` field.

## Update Grouping bootstrap data
Ensure bootstrapping data has a good variety of labels including a few each
with the labels "main", "drink", "dessert", "appetizer", "breakfast", and
"side", and a couple with none of those labels.
