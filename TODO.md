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


## Add `Type` attribute to Label
Add a string field to the Label database table and struct to hold a meta-label
describing what category of label this is. (for example, "vegan" would be a
"dietary restriction", while "mexican" would be a "cuisine" and "soup" would be
a "dish type".

Include this new field in the `addLabel` handler and the `createLabel` model
method.

Add a new `updateLabelType` method to the model to update a label's type in the
database.

Add a new `editLabelType` handler to the privileged (or admin, if it exists)
router that calls the `updateLabelType` model method

Add a PUT route `/label/{label_name}/type` handled by `editLabelType` to the
privileged (or admin, if it exists) router.

Add the type field to `bootstrapping/labels.csv` and provide a value for all
the existing labels. Mapping of label_ids to types to be provided when we make
the change.

Generate a MySQL query for altering the production labels table to include the
`type` field and populate it the same as we do for bootstrapping.

## Add `Icon` attribute to Label
Add a nullable field to the Label database table and struct to hold a single
character (likely an emoji or other multibyte code point) to represent this
label.

Include this new field in the `addLabel` handler and the `createLabel` model
method.

Add a new `updateLabelIcon` method to the model to update a label's icon in the
database.

Add a new `editLabelIcon` handler to the privileged (or admin, if it exists)
router that calls the `updateLabelIcon` model method

Add a PUT route `/label/{label_name}/icon` handled by `editLabelIcon` to the
privileged (or admin, if it exists) router.

Add the icon field to `bootstrapping/labels.csv`. Only provide an icon in the
bootstrapping CSV for labels with icons provided here (to be added before
implementation)

Generate a MySQL query for altering the production labels table to include the
`icon` field and populate it the same as we do for bootstrapping.

## Add support for Recipe's `New` field
Add a `setRecipeNewFlag` model method that sets the `new` field on the recipe
to a passed value.

Add new handlers `flagRecipeCooked` and `unFlagRecipeCooked` to the privileged
(or admin, if it exists) router that calls the `setRecipeNewFlag` model method.

Add new PUT routes `/recipe/{id}/mark_cooked` and `/recipe/{id}/mark_new`
handled by `flagRecipeCooked` and `unFlagRecipeCooked` to
the privileged (or admin, if it exists) router.

Update a few of the recipes in `bootstrapping/recipes.csv` to have the `new`
flag toggled on.

## Update Grouping bootstrap data
Ensure bootstrapping data has a good variety of labels including a few each
with the labels "main", "drink", "dessert", "appetizer", "breakfast", and
"side", and a couple with none of those labels.
