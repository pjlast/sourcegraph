import { gql, useQuery } from '@sourcegraph/http-client'

import { OrgFeatureFlagOverridesResult, OrgFeatureFlagOverridesVariables } from '../graphql-operations'

// A union of all feature flags we currently have.
// If there are no feature flags at the moment, this should be `never`.
export type FeatureFlagName =
    | 'quick-start-tour-for-authenticated-users'
    | 'new-repo-page'
    | 'insight-polling-enabled'
    | 'ab-visitor-tour-with-notebooks'
    | 'ab-email-verification-alert'
    | 'contrast-compliant-syntax-highlighting'
    | 'admin-analytics-disabled'
    | 'admin-analytics-cache-disabled'
    | 'search-input-show-history'
    | 'user-management-disabled'
    | 'enable-streaming-git-blame'

interface OrgFlagOverride {
    orgID: string
    flagName: string
    value: boolean
}

/**
 * // TODO: clarify why to use this if GQL already takes care of overrides?
 * Fetches all feature flag overrides for organizations that the current user is a member of
 */
export function useFlagsOverrides(): { data: OrgFlagOverride[]; loading: boolean } {
    const { data, loading } = useQuery<OrgFeatureFlagOverridesResult, OrgFeatureFlagOverridesVariables>(
        gql`
            query OrgFeatureFlagOverrides {
                organizationFeatureFlagOverrides {
                    namespace {
                        id
                    }
                    targetFlag {
                        ... on FeatureFlagBoolean {
                            name
                        }
                        ... on FeatureFlagRollout {
                            name
                        }
                    }
                    value
                }
            }
        `,
        { fetchPolicy: 'cache-and-network' }
    )

    if (!data) {
        return { data: [], loading }
    }

    return {
        data: data?.organizationFeatureFlagOverrides.map(value => ({
            orgID: value.namespace.id,
            flagName: value.targetFlag.name,
            value: value.value,
        })),
        loading,
    }
}
