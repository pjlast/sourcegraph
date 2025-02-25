import React from 'react'

import classNames from 'classnames'
import ChevronLeftIcon from 'mdi-react/ChevronLeftIcon'
import ChevronRightIcon from 'mdi-react/ChevronRightIcon'
import PageFirstIcon from 'mdi-react/PageFirstIcon'
import PageLastIcon from 'mdi-react/PageLastIcon'

import { Button } from '../Button'
import { Icon } from '../Icon'
import { Tooltip } from '../Tooltip'
import { Text } from '../Typography'

import styles from './PageSwitcher.module.scss'

export interface PageSwitcherProps {
    totalLabel?: string
    totalCount: null | number
    hasNextPage: null | boolean
    hasPreviousPage: null | boolean
    goToNextPage: () => void
    goToPreviousPage: () => void
    goToFirstPage: () => void
    goToLastPage: () => void
    className?: string
}

/**
 * PageSwitcher is used to render pagination control for a cursor-based
 * pagination.
 *
 * It works best together with the `usePageSwitcherPagination` hook and
 * is our recommended way of implementing pagination.
 */
export const PageSwitcher: React.FunctionComponent<React.PropsWithChildren<PageSwitcherProps>> = props => {
    const {
        className,
        totalLabel,
        totalCount,
        goToFirstPage,
        goToPreviousPage,
        goToNextPage,
        goToLastPage,
        hasPreviousPage,
        hasNextPage,
    } = props

    return (
        <nav className={className}>
            <ul className={styles.list}>
                <li>
                    <Tooltip content="First page">
                        <Button
                            aria-label="Go to first page"
                            className={classNames(styles.button, 'mx-3')}
                            variant="secondary"
                            outline={true}
                            disabled={hasPreviousPage !== null ? !hasPreviousPage : true}
                            onClick={goToFirstPage}
                        >
                            <Icon aria-hidden={true} as={PageFirstIcon} className={styles.firstPageButton} />
                        </Button>
                    </Tooltip>
                </li>
                <li>
                    <Button
                        className={classNames(styles.button, styles.previousButton, 'mx-1')}
                        aria-label="Go to previous page"
                        variant="secondary"
                        outline={true}
                        disabled={hasPreviousPage !== null ? !hasPreviousPage : true}
                        onClick={goToPreviousPage}
                    >
                        <Icon
                            aria-hidden={true}
                            as={ChevronLeftIcon}
                            className={classNames('mr-1', styles.previousButtonIcon)}
                        />
                        <span className={styles.previousButtonLabel}>Prev</span>
                    </Button>
                </li>
                <li>
                    <Button
                        className={classNames(styles.button, styles.nextButton, 'mx-1')}
                        aria-label="Go to next page"
                        variant="secondary"
                        outline={true}
                        disabled={hasNextPage !== null ? !hasNextPage : true}
                        onClick={goToNextPage}
                    >
                        <span className={styles.nextButtonLabel}>Next</span>
                        <Icon
                            aria-hidden={true}
                            as={ChevronRightIcon}
                            className={classNames('ml-1', styles.nextButtonIcon)}
                        />
                    </Button>
                </li>
                <li>
                    <Tooltip content="Last page">
                        <Button
                            aria-label="Go to last page"
                            className={classNames(styles.button, 'mx-3')}
                            variant="secondary"
                            outline={true}
                            disabled={hasNextPage !== null ? !hasNextPage : true}
                            onClick={goToLastPage}
                        >
                            <Icon aria-hidden={true} as={PageLastIcon} className={styles.lastPageButton} />
                        </Button>
                    </Tooltip>
                </li>
            </ul>
            {totalCount !== null && totalLabel !== undefined ? (
                <div className={styles.label}>
                    <Text className="text-muted" size="small">
                        Total{' '}
                        <Text weight="bold" as="strong">
                            {totalLabel}
                        </Text>
                        : {totalCount}
                    </Text>
                </div>
            ) : null}
        </nav>
    )
}
