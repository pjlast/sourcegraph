# Code host connections

Sourcegraph can sync repositories from code hosts and other similar services. We designate code hosts between Tier 1 and Tier 2 based on Sourcegraph's capabilities when used with those code hosts. 

## Tier 1 code hosts

Tier 1 code hosts are our highest level of support for code hosts. When leveraging a Tier 1 code host, you can expect:

- Scalable repository syncing - Sourcegraph is able to reliably sync repositories from this code host up to 100k repositories. (SLA TBD)
- Scalable permissions syncing - Sourcegraph is able to reliably sync permissions from this code host for up to 10k users. (SLA TBD)
- Authentication - Sourcegraph is able to leverage authentication from this code host (i.e. Login with GitHub). 
- Code Search - A developer can seamlessly search across repositories from this code host. (SLAs TBD)
- Code Monitors - A developer can create a code monitor to monitor code in this repository. 
- Code Insights - Code Insights reliably works on code sync'd from a tier 1 code host.
- Batch Changes - A Batch Change can be leveraged to submit code changes back to a tier 1 code host while respecting code host permissions.

<table>
   <thead>
      <tr>
        <th>Code Host</th>
        <th>Status</th>
        <th>Repository Syncing</th>
        <th>Permissions Syncing</th>
        <th>Authentication</th>
        <th>Code Search</th>
        <th>Code Monitors</th>
        <th>Code Insights</th>
        <th>Batch Changes</th>
      </tr>
   </thead>
   <tbody>
      <tr>
        <td>GitHub.com</td>
        <td>Tier 1</td>
        <td class="indexer-implemented-y">✓</td> <!-- Repository Syncing -->
        <td class="indexer-implemented-y">✓</td> <!-- Permissions Syncing -->
        <td class="indexer-implemented-y">✓</td> <!-- Authentication -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Search -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Monitors -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Insights -->
        <td class="indexer-implemented-y">✓</td> <!-- Batch Changes -->
      </tr>
      <tr>
        <td>GitHub Self-Hosted Enterprise</td>
        <td>Tier 1</td>
        <td class="indexer-implemented-y">✓</td> <!-- Repository Syncing -->
        <td class="indexer-implemented-y">✓</td> <!-- Permissions Syncing -->
        <td class="indexer-implemented-y">✓</td> <!-- Authentication -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Search -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Monitors -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Insights -->
        <td class="indexer-implemented-y">✓</td> <!-- Batch Changes -->
      </tr>
      <tr>
        <td>GitLab.com</td>
        <td>Tier 1</td>
        <td class="indexer-implemented-y">✓</td> <!-- Repository Syncing -->
        <td class="indexer-implemented-y">✓</td> <!-- Permissions Syncing -->
        <td class="indexer-implemented-y">✓</td> <!-- Authentication -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Search -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Monitors -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Insights -->
        <td class="indexer-implemented-y">✓</td> <!-- Batch Changes -->
      </tr>
      <tr>
        <td>GitLab Self-Hosted</td>
        <td>Tier 1</td>
        <td class="indexer-implemented-y">✓</td> <!-- Repository Syncing -->
        <td class="indexer-implemented-y">✓</td> <!-- Permissions Syncing -->
        <td class="indexer-implemented-y">✓</td> <!-- Authentication -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Search -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Monitors -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Insights -->
        <td class="indexer-implemented-y">✓</td> <!-- Batch Changes -->
      </tr>
      <tr>
        <td>BitBucket Cloud</td>
        <td>Tier 1</td>
        <td class="indexer-implemented-y">✓</td> <!-- Repository Syncing -->
        <td class="indexer-implemented-n">✗</td> <!-- Permissions Syncing -->
        <td class="indexer-implemented-n">✗</td> <!-- Authentication -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Search -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Monitors -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Insights -->
        <td class="indexer-implemented-y">✓</td> <!-- Batch Changes -->
      </tr>
      <tr>
        <td>BitBucket Server</td>
        <td>Tier 1</td>
        <td class="indexer-implemented-y">✓</td> <!-- Repository Syncing -->
        <td class="indexer-implemented-y">✓</td> <!-- Permissions Syncing -->
        <td class="indexer-implemented-y">✓</td> <!-- Authentication -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Search -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Monitors -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Insights -->
        <td class="indexer-implemented-y">✓</td> <!-- Batch Changes -->
      </tr>
      <tr>
        <td>Perforce</td>
        <td>Tier 2 (Working on Tier 1)</td>
        <td class="indexer-implemented-y">✓</td> <!-- Repository Syncing -->
        <td class="indexer-implemented-y">✓</td> <!-- Permissions Syncing -->
        <td class="indexer-implemented-n">✗</td> <!-- Authentication -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Search -->
        <td class="indexer-implemented-y">✓</td> <!-- Code Monitors -->
        <td class="indexer-implemented-n">✗</td> <!-- Code Insights -->
        <td class="indexer-implemented-n">✗</td> <!-- Batch Changes -->
      </tr>
   </tbody>
</table>

#### Status definitions

An code host status is:

- 🟢 _Generally Available:_ Available as a normal product feature up to 100k repositories.
- 🟡 _Partially available:_ Available, but may be limited in some significant ways (either missing or buggy functionality). If you plan to leverage this, please contact your Customer Engineer. 
- 🔴 _Not available:_ This functionality is not available within Sourcegraph

## Tier 2: Code Hosts
We recognize there are other code hosts including CVS, Azure Dev Ops, SVN, and many more. Today, we do not offer native integrations with these code hosts and customers are advised to leverage [Src-srv-git](./non-git.md) and the [explicit permissions API](../repo/permissions.md#explicit-permissions-api) as a way to ingest code and permissions respectively into Sourcegraph. 

[Src-srv-git](./non-git.md) and the [explicit permissions API](../repo/permissions.md#explicit-permissions-api) follow the same scale guidance shared above (up to 100k repos and 10k users). 


## Full Code Host Docs

**Site admins** can configure the following code hosts:

- [GitHub](github.md)
- [GitLab](gitlab.md)
- [Bitbucket Cloud](bitbucket_cloud.md)
- [Bitbucket Server / Bitbucket Data Center](bitbucket_server.md)
- [Other Git code hosts (using a Git URL)](other.md)
- [Non-Git code hosts](non-git.md)
  - [Perforce](../repo/perforce.md)
  - [JVM dependencies](jvm.md)
  - [Go dependencies](go.md)
  - [npm dependencies](npm.md)
  - [Python dependencies](python.md)
  - [Ruby dependencies](ruby.md)

**Users** can configure the following public code hosts:

- [GitHub.com](github.md)
- [GitLab.com](gitlab.md)


## Rate limits

Sourcegraph makes our best effort to use the least amount of calls to your code host. However, it is possible for Sourcegraph 
to encounter rate limits in some scenarios. Please see the specific code host documentation for more information and how to 
mitigate these issues. 

### Rate limit syncing

Sourcegraph has a mechanism of syncing code host rate limits. When Sourcegraph is started, code host configurations of all
external services are checked for rate limits and these rate limits are stored and used.

When any of code host configurations is edited, rate limits are synchronized and updated if needed, this way Sourcegraph always 
knows how many requests to which code host can be sent at a given point of time.

### Current rate limit settings

Current rate limit settings can be viewed by site admins on the following page: `Site Admin -> Instrumentation -> Repo Updater -> Rate Limiter State`.
This page includes rate limit settings for all external services configured in Sourcegraph. 

Here is an example of one external service, including information about external service name,  maximum allowed burst of requests,
maximum allowed requests per second and whether the limiter is infinite (there is no rate limiting):
```json
{
  "extsvc:github:4": {
    "Burst": 10,
    "Limit": 1.3888888888888888,
    "Infinite": false
  }
}
```

### Increasing code host rate limits

Customers should avoid creating additional **free** accounts for the purpose of circumventing code-host rate limits. 
Some code hosts have higher rate limits for **paid** accounts and allow the creation of additional **paid** accounts which 
Sourcegraph can leverage.

Please contact support@sourcegraph.com if you encounter rate limits.
