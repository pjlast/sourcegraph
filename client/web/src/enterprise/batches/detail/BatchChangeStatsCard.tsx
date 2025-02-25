import React from 'react'

import { VisuallyHidden } from '@reach/visually-hidden'
import classNames from 'classnames'
import CheckCircleOutlineIcon from 'mdi-react/CheckCircleOutlineIcon'
import ProgressCheckIcon from 'mdi-react/ProgressCheckIcon'

import { pluralize } from '@sourcegraph/common'
import { Badge, Icon, Heading, H3, H4 } from '@sourcegraph/wildcard'

import { DiffStatStack } from '../../../components/diff/DiffStat'
import { BatchChangeFields } from '../../../graphql-operations'
import { BatchChangeStatePill } from '../list/BatchChangeStatePill'

import {
    ChangesetStatusUnpublished,
    ChangesetStatusOpen,
    ChangesetStatusClosed,
    ChangesetStatusMerged,
    ChangesetStatusDraft,
    ChangesetStatusArchived,
} from './changesets/ChangesetStatusCell'

import styles from './BatchChangeStatsCard.module.scss'

interface BatchChangeStatsCardProps {
    batchChange: Pick<BatchChangeFields, 'diffStat' | 'changesetsStats' | 'state'>
    className?: string
}

// Rounds percent down to the nearest integer (you don't say 1/50/100% complete until at
// least 1/50/100% is actually completed).
const formatDisplayPercent = (percent: number): string => `${Math.floor(percent)}%`

export const BatchChangeStatsCard: React.FunctionComponent<React.PropsWithChildren<BatchChangeStatsCardProps>> = ({
    batchChange,
    className,
}) => {
    const { changesetsStats: stats, diffStat } = batchChange
    const percentComplete = stats.total === 0 ? 0 : ((stats.closed + stats.merged + stats.deleted) / stats.total) * 100
    const isCompleted = stats.closed + stats.merged + stats.deleted === stats.total
    let BatchChangeStatusIcon = ProgressCheckIcon
    if (isCompleted) {
        BatchChangeStatusIcon = CheckCircleOutlineIcon
    }
    return (
        <div className={classNames(className)}>
            <div className="d-flex flex-wrap align-items-center flex-grow-1">
                {/*
                    a11y-ignore
                    Rule: "color-contrast" (Elements must have sufficient color contrast)
                    GitHub issue: https://github.com/sourcegraph/sourcegraph/issues/33343
                */}
                <BatchChangeStatePill
                    state={batchChange.state}
                    className={classNames('a11y-ignore', styles.batchChangeStatsCardStateBadge)}
                />
                <div className={classNames(styles.batchChangeStatsCardDivider, 'mx-3')} />
                <div className="d-flex align-items-center">
                    <Heading as="h3" styleAs="h1" className="d-inline mb-0" aria-hidden="true">
                        <Icon
                            className={classNames('mr-2', isCompleted ? 'text-success' : 'text-muted')}
                            as={BatchChangeStatusIcon}
                            aria-hidden={true}
                        />
                    </Heading>{' '}
                    <span className={classNames(styles.batchChangeStatsCardCompleteness, 'lead text-nowrap')}>
                        {`${formatDisplayPercent(percentComplete)} complete`}
                    </span>
                </div>
                <div className={classNames(styles.batchChangeStatsCardDivider, 'd-none d-md-block mx-3')} />
                <DiffStatStack className={styles.batchChangeStatsCardDiffStat} {...diffStat} />
                <div className="d-flex flex-wrap justify-content-end flex-grow-1">
                    <BatchChangeStatsTotalAction count={stats.total} />
                    <ChangesetStatusUnpublished
                        label={
                            <H4 className="font-weight-normal text-muted m-0">
                                {stats.unpublished}{' '}
                                <VisuallyHidden>{pluralize('changeset', stats.unpublished)}</VisuallyHidden> unpublished
                            </H4>
                        }
                        className={classNames(styles.batchChangeStatsCardStat, 'd-flex flex-grow-0 px-2 text-truncate')}
                    />
                    <ChangesetStatusDraft
                        label={
                            <H4 className="font-weight-normal text-muted m-0">
                                {stats.draft} <VisuallyHidden>{pluralize('changeset', stats.draft)}</VisuallyHidden>{' '}
                                draft
                            </H4>
                        }
                        className={classNames(styles.batchChangeStatsCardStat, 'd-flex flex-grow-0 px-2 text-truncate')}
                    />
                    <ChangesetStatusOpen
                        label={
                            <H4 className="font-weight-normal text-muted m-0">
                                {stats.open} <VisuallyHidden>{pluralize('changeset', stats.open)}</VisuallyHidden> open
                            </H4>
                        }
                        className={classNames(styles.batchChangeStatsCardStat, 'd-flex flex-grow-0 px-2 text-truncate')}
                    />
                    <ChangesetStatusClosed
                        label={
                            <H4 className="font-weight-normal text-muted m-0">
                                {stats.closed} <VisuallyHidden>{pluralize('changeset', stats.closed)}</VisuallyHidden>{' '}
                                closed
                            </H4>
                        }
                        className={classNames(styles.batchChangeStatsCardStat, 'd-flex flex-grow-0 px-2 text-truncate')}
                    />
                    <ChangesetStatusMerged
                        label={
                            <H4 className="font-weight-normal text-muted m-0">
                                {stats.merged} <VisuallyHidden>{pluralize('changeset', stats.merged)}</VisuallyHidden>{' '}
                                merged
                            </H4>
                        }
                        className={classNames(styles.batchChangeStatsCardStat, 'd-flex flex-grow-0 pl-2 text-truncate')}
                    />
                    <ChangesetStatusArchived
                        label={
                            <H4 className="font-weight-normal text-muted m-0">
                                {stats.archived}{' '}
                                <VisuallyHidden>{pluralize('changeset', stats.archived)}</VisuallyHidden> archived
                            </H4>
                        }
                        className={classNames(styles.batchChangeStatsCardStat, 'd-flex flex-grow-0 pl-2 text-truncate')}
                    />
                </div>
            </div>
        </div>
    )
}

export const BatchChangeStatsTotalAction: React.FunctionComponent<React.PropsWithChildren<{ count: number }>> = ({
    count,
}) => (
    <H4
        as={H3}
        className={classNames(
            styles.batchChangeStatsCardStat,
            'font-weight-normal m-0 flex-grow-0 pr-2 text-truncate text-nowrap d-flex flex-column align-items-center justify-content-center'
        )}
        aria-label={`${count} total ${pluralize('changeset', count)}`}
    >
        <span className={styles.batchChangeStatsCardChangesetsPill}>
            <Badge variant="secondary" pill={true}>
                {count}
            </Badge>
        </span>
        <span className="text-muted">{pluralize('changeset', count)}</span>
    </H4>
)
