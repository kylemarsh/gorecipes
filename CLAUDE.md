# Overview
This is the backend code for a recipe database. It provides an API wrapper for
a relational database -- MySQL in production, and MySQL or sqlite in
development. It is written in golang and deployed to a private server via
github actions upon push to specific branches.

# Code Structure
The code is mostly flat.

There is a `.github/workflows` directory that configures workflows for github
to perform on push

There is a `bootstrapping` folder that holds a program
to initialize a new database and several CSV files that contain data to be
imported into the database to initialize it.

There is a `Makefile` that defines several build commands, mostly used by the
deployment workflows.

There is a `README.md` which describes how to develop, what the configuration
options are, and provides example requests that you can make against the
development server.

`mem.config`, `dev.config`, and `dev_mysql.config` are JSON configuration files for
setting up the server for development. (There is a `gorecipes.conf` on the
deploy host for production) These specify values for the configuration options
described in `README.md`. The configuration includes `Debug`, `DbDialect`, `DbDSN`,
`JwtSecret`, and `Origins` (required when not in debug mode). The primary difference
between the dev configs is that `mem.config` sets the server up to use an in-memory
sqlite3 database, `dev.config` sets it up to read from a sqlite database stored on
disk as `recipes_sqlite.db`, and `dev_mysql.config` sets it up to read from a mysql
server hosted on the local machine.

The entry point is `main.go`, which handles initialization, configuration,
defining the routers, and launching the http server.

`bootstrap.go` is the same logic as is found in
`bootstrapping/bootstrap_recipes.go`, but packaged to be callable within the
main `gorecipes` code. This allows us to launch the server with a fresh
in-memory sqlite database and bootstrap it in one command without needing an
on-disk artifact.

`debug.go`, `public.go` and `privileged.go` hold functions that map to routes
in the http server. These are primarily ways of interacting with the database.

`util.go` holds helper functions

`model.go` defines structures for the data, where an instance of a structure
represent one record from the corresponding database table. We have structures
for `User`, `Recipe`, `Label`, and `Note`. This file also provides functions
for interacting directly with the database via either the mysql driver or the
sqlite3 driver.

# Libraries
This project uses the golang standard libraries. Additional libraries are
listed in the file `go.mod`. Of importance are:
 - The project uses gorilla/mux for routing
 - The project uses jmoiron/sqlx for database abstraction
 - The project uses golang-jwt/jwt for authentication

# Database Model
There are 4 types of record tracked, each with its own database table and
struct:
 - `User`: represents a person who is authorized to view (and maybe modify) the
   recipe database
 - `Recipe`: The core representation of a recipe itself.
 - `Label`: A taxonomic tag for recipes.
 - `Note`: A note attached to a recipe.

In addition to one table for each of these structures, there is a junction
table, `recipe_label`, that provides a many-to-many mapping between recipes
and labels.

## User
A `User` has the following attributes:
 - `ID` (`user_id` in the db): the primary key for this user in the database
 - `Username`: the string the user will use to log in
 - `HashedPassword` (`password` in the db): hash of the user's password
 - `PlaintextPassword` (`plaintext_pw_bootstrapping_only` in the db): the
   user's password in plain text. Only used for bootstrapping development db

## Recipe
A `Recipe` has the following attributes:
 - `ID` (`recipe_id` in the db): the primary key for this recipe in the db
 - `Title`: the recipe's title, displayed in a search list
 - `Body` (`recipe_body` in the db): the builk of the recipe as a free text
   field. This usually includes ingredients and instructions both
 - `Time` (`total_time` in the db): how long this recipe takes to cook
 - `ActiveTime` (`active_time` in the db): how long the cook needs to spend
   working on this this recipe (chopping, stirring, etc.)
 - `Deleted`: boolean to mark a recipe as deleted. Deleted recipes can be
   restored by setting this back to `false`
 - `New`: boolean to mark a recipe as new. Once cooked we mark it `false`.
 - `Labels`: array of `Label` structures this recipe is tagged with
 - `Notes`: array of `Note` structures attached to this recipe


## Label
A `Label` has the following attributes:
 - `ID` (`label_id` in the db): the primary key for this label in the db
 - `Label`: the label's name

 Labels are many-to-many with recipes, so they need a junction table. A label
 is some kind of informative tag to use for filtering recipes. Examples are:
  - "chicken"
  - "beef"
  - "soup"
  - "dessert"
  - "mexican"
  - "thai"
  - "GlutenFree"
  - "vegan"

## Note
A `Note` has the following attributes:
 - `ID` (`note_id` in the db): the primary key for this note in the db
 - `RecipeId` (`recipe_id` in the db): the PK for the recipe
 - `Created` (`create_date` in the db): the unix timestamp of the date this
   note was created (used for sorting the notes)
 - `Note`: the text body of the note
 - `Flagged`: boolean to mark a note as incorporated into the recipe

 The intended pattern is to put a note on the recipe with potential updates to
 cooking times, temperatures, or ingredients. Then at a later time the note can
 be flagged to indicate that the recipe was actually updated with its contents.

# Routing
The main class defines three routers:
 - `router` handles unauthenticated requests - logging in, fetching recipe
   titles and labels
 - `privRouter` handles requests requiring authenticatin - fetching recipe
   details, adding/editing/deleting records, and marking recipes as new/cooked.
 - `debugRouter` handles special debugging requests and is only accesible when
   the server is running with the `debug` configuration equal to `true`

The routers are `mux` routers from `github.com/gorilla/mux` and routes are set
up by calling `Handle` on the router:
 - The first argument is the path to route. `{}` in the route indicate
   parameters to pass to the handling function.
 - The second argument is the function to call with requests to this route.
 - Handle can be chained with `Method` which is passed the HTTP methods allowed
   for this route.

The `privRouter` and `debugRouter` use the `authRequired` and `debugRequired`
middlewares, respectively, to enforce protections around their routes.

## Recipe New Flag Routes
The `privRouter` includes two routes for managing the `new` flag on recipes:
 - `PUT /priv/recipe/{id}/mark_cooked` - handled by `flagRecipeCooked()` in
   `privileged.go`, sets the recipe's `new` field to `false` to indicate it has
   been cooked
 - `PUT /priv/recipe/{id}/mark_new` - handled by `unFlagRecipeCooked()` in
   `privileged.go`, sets the recipe's `new` field to `true` to mark it as
   new/uncooked

Both handlers validate the recipe ID, verify the recipe exists (returning 404
if not), and call `setRecipeNewFlag()` in `model.go` to update the database.
They follow the same pattern as `flagNote()`/`unFlagNote()` for consistency.

# Development
Use the instructions in README.md to launch a devlopment server for testing
changes.

# Deploying
Staging: Pushes to the `staging` branch automatically build and deploy to the
staging server with a freshly bootstrapped database.

Production: Pushes to the `main` branch automatically build and deploy to the
production server using the production mysql database

# Future Work
See TODO.md for descriptions of features to be added

# Making Changes
##Before making any changes
 1. Explore the repository structure
 2. Identify relevant files
 3. Explain the current implementation
 4. Propose changes. Do not write code until the user signs-off on the plan


##When making changes
 - ALWAYS create a new branch for the feature of bugfix with a descriptive name
 - NEVER develop directly on the `main` branch
 - NEVER merge feature branches in to `main`
 - NEVER push to the remote repository
 - Prefer to Follow existing patterns in the repository. Do not introduce new
   frameworks or patterns unless asked to do so. If you think a different
   pattern or framework is the best way to accomplish something, ask the user
   whether you should use it or not, explain why this pattern is correct, and
   whether or not existing code should be updated to match
 - Once on the correct feature branch, BEGIN by proposing new or updated tests
 - Ensure any new routes are added to the appropriate router
 - Ensure the CSV files in `bootstrapping/` reflect the new structure of the
   model and contain records that populate any new fields.
 - Run through edge cases
 - Verify imports
 - Check for compilation errors
 - Confirm tests compile and pass

##After making a change
Once the user has accepted changes:

 1. Explore the repository structure again
 2. Identify changes made; DO NOT assume that the changes made are exactly what
    was discussed in the current context. Look at the diff between the feature
    branch and `main`.
 3. Update this document and any other documentation to reflect the new
    structure of the project
 4. Update the TODO.md file to remove the feature request

