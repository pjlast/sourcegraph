import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'

import { dataOrThrowErrors, gql } from '@sourcegraph/http-client'

import { requestGraphQL } from '../../backend/graphql'
import {
    BatchSpecsVariables,
    BatchSpecsResult,
    Scalars,
    BatchChangeBatchSpecsVariables,
    BatchChangeBatchSpecsResult,
    BatchSpecListConnectionFields,
} from '../../graphql-operations'

export const queryBatchSpecs = ({
    first,
    after,
    includeLocallyExecutedSpecs,
    excludeEmptySpecs,
}: BatchSpecsVariables): Observable<BatchSpecListConnectionFields> =>
    requestGraphQL<BatchSpecsResult, BatchSpecsVariables>(
        gql`
            query BatchSpecs(
                $first: Int
                $after: String
                $includeLocallyExecutedSpecs: Boolean
                $excludeEmptySpecs: Boolean
            ) {
                batchSpecs(
                    first: $first
                    after: $after
                    includeLocallyExecutedSpecs: $includeLocallyExecutedSpecs
                    excludeEmptySpecs: $excludeEmptySpecs
                ) {
                    ...BatchSpecListConnectionFields
                }
            }

            ${BATCH_SPEC_LIST_CONNECTION_FIELDS}
        `,
        {
            first,
            after,
            includeLocallyExecutedSpecs,
            excludeEmptySpecs,
        }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.batchSpecs)
    )

export const queryBatchChangeBatchSpecs = (id: Scalars['ID']) => ({
    first,
    after,
    includeLocallyExecutedSpecs,
    excludeEmptySpecs,
}: Omit<BatchChangeBatchSpecsVariables, 'id'>): Observable<BatchSpecListConnectionFields> =>
    requestGraphQL<BatchChangeBatchSpecsResult, BatchChangeBatchSpecsVariables>(
        gql`
            query BatchChangeBatchSpecs(
                $id: ID!
                $first: Int
                $after: String
                $includeLocallyExecutedSpecs: Boolean
                $excludeEmptySpecs: Boolean
            ) {
                node(id: $id) {
                    __typename
                    ... on BatchChange {
                        batchSpecs(
                            first: $first
                            after: $after
                            includeLocallyExecutedSpecs: $includeLocallyExecutedSpecs
                            excludeEmptySpecs: $excludeEmptySpecs
                        ) {
                            ...BatchSpecListConnectionFields
                        }
                    }
                }
            }

            ${BATCH_SPEC_LIST_CONNECTION_FIELDS}
        `,
        {
            id,
            first,
            after,
            includeLocallyExecutedSpecs,
            excludeEmptySpecs,
        }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.node) {
                throw new Error('Batch change not found')
            }
            if (data.node.__typename !== 'BatchChange') {
                throw new Error(`Node is a ${data.node.__typename}, not a BatchChange`)
            }
            return data.node.batchSpecs
        })
    )

const PARTIAL_BATCH_WORKSPACE_FILE_FIELDS = gql`
    fragment PartialBatchSpecWorkspaceFileFields on BatchSpecWorkspaceFile {
        id
        name
        binary
        byteSize
    }
`

const BATCH_WORKSPACE_FILE_FIELDS = gql`
    fragment BatchSpecWorkspaceFileFields on BatchSpecWorkspaceFile {
        ...PartialBatchSpecWorkspaceFileFields
        highlight(disableTimeout: false) {
            aborted
            html
        }
    }

    ${PARTIAL_BATCH_WORKSPACE_FILE_FIELDS}
`

const BATCH_SPEC_LIST_FIELDS_FRAGMENT = gql`
    fragment BatchSpecListFields on BatchSpec {
        __typename
        id
        state
        startedAt
        finishedAt
        createdAt
        source
        description {
            __typename
            name
        }
        namespace {
            namespaceName
            url
        }
        creator {
            username
        }
        originalInput
        files {
            totalCount
            pageInfo {
                endCursor
                hasNextPage
            }
            nodes {
                ...PartialBatchSpecWorkspaceFileFields
            }
        }
    }

    ${PARTIAL_BATCH_WORKSPACE_FILE_FIELDS}
`

const BATCH_SPEC_LIST_CONNECTION_FIELDS = gql`
    fragment BatchSpecListConnectionFields on BatchSpecConnection {
        totalCount
        pageInfo {
            endCursor
            hasNextPage
        }
        nodes {
            ...BatchSpecListFields
        }
    }

    ${BATCH_SPEC_LIST_FIELDS_FRAGMENT}
`

export const BATCH_SPEC_WORKSPACE_FILE = gql`
    query BatchSpecWorkspaceFile($id: ID!) {
        node(id: $id) {
            ... on BatchSpecWorkspaceFile {
                ...BatchSpecWorkspaceFileFields
            }
        }
    }

    ${BATCH_WORKSPACE_FILE_FIELDS}
`

const BATCH_CHANGE_FILE_BASE_URL = '/.api/files/batch-changes'

export const generateFileDownloadLink = async (specId: string, fileId: string): Promise<string> => {
    const url = `${BATCH_CHANGE_FILE_BASE_URL}/${specId}/${fileId}`
    const file = await fetch(url, {
        headers: {
            ...window.context.xhrHeaders,
        },
    })
    const fileBlob = await file.blob()
    return URL.createObjectURL(fileBlob)
}
