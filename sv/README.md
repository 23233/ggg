# simple_valid as sv
a simple paramater validator for iris , use middleware

## use see test 
```cassandraql
// Define req valid struct
	type req struct {
		Name string `json:"name" url:"name" comment:"name" validate:"required"`
	}

// Use middleware
	app.Get("/", sv.Run(new(req)), func(ctx iris.Context) {
		req := ctx.Values().Get("sv").(*req) // <- this get req data 
		_, _ = ctx.JSON(iris.Map{"name": req.Name})
	})
```
