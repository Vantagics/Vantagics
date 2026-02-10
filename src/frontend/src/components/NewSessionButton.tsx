import React from 'react';
import './NewSessionButton.css';
import { useLanguage } from '../i18n';

/**
 * Props interface for NewSessionButton component
 * Requirements: 2.3
 */
export interface NewSessionButtonProps {
    onClick: () => void;
}

/**
 * NewSessionButton component
 *
 * Provides a prominent button to create new analysis sessions.
 * Positioned at the top of the LeftPanel.
 *
 * - Full-width button with icon and text
 * - Always enabled (no dependency on data source selection)
 *
 * Requirements: 2.3
 */
const NewSessionButton: React.FC<NewSessionButtonProps> = ({
    onClick,
}) => {
    const { t } = useLanguage();
    const tooltipText = t('create_new_session');

    return (
        <div className="new-session-button-container">
            <button
                className="new-session-button"
                onClick={onClick}
                title={tooltipText}
                aria-label={tooltipText}
            >
                <span className="new-session-icon" aria-hidden="true">+</span>
                <span className="new-session-text">{t('new_session')}</span>
            </button>
        </div>
    );
};

export default NewSessionButton;
