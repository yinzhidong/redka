import { RedisClient } from "bun";

const client = new RedisClient("redis://localhost:6379/0", {
    // autoReconnect: true,
    // connectionTimeout: 10000,
    // maxRetries: 2,
});
const init = await client.set("counter", "0");
console.log("init===", init)


const server = Bun.serve({
  // `routes` requires Bun v1.2.3+
  routes: {
    // Static routes
    "/api/status": new Response("OK"),

    "/redis": async req => {
        await client.incr("counter");
        const result = await client.get("counter");
        return new Response(`Redis ${result}`);
    },

    // Dynamic routes
    "/users/:id": req => {
      return new Response(`Hello User ${req.params.id}!`);
    },


    // Per-HTTP method handlers
    "/api/posts": {
      GET: () => new Response("List posts"),
      POST: async req => {
        const body = await req.json();
        return Response.json({ created: true, ...body });
      },
    },

    // Wildcard route for all routes that start with "/api/" and aren't otherwise matched
    "/api/*": Response.json({ message: "Not found" }, { status: 404 }),

    // Redirect from /blog/hello to /blog/hello/world
    // "/blog/hello": Response.redirect("/blog/hello/world"),

    // Serve a file by lazily loading it into memory
    // "/favicon.ico": Bun.file("./favicon.ico"),
  },

  // (optional) fallback for unmatched routes:
  // Required if Bun's version < 1.2.3
  fetch(req) {
    return new Response("Not Found", { status: 404 });
  },
});


console.log(`Server running at ${server.url}`);