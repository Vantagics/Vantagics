import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, vi } from 'vitest';
import NewSessionButton, { NewSessionButtonProps } from './NewSessionButton';

describe('NewSessionButton Component', () => {
    const defaultProps: NewSessionButtonProps = {
        onClick: vi.fn(),
    };

    describe('Rendering', () => {
        it('should render the button with icon and text', () => {
            render(<NewSessionButton {...defaultProps} />);
            const button = screen.getByRole('button');
            expect(button).toBeInTheDocument();
            expect(screen.getByText('+')).toBeInTheDocument();
            expect(screen.getByText('New Session')).toBeInTheDocument();
        });

        it('should render within a container div', () => {
            const { container } = render(<NewSessionButton {...defaultProps} />);
            const containerDiv = container.querySelector('.new-session-button-container');
            expect(containerDiv).toBeInTheDocument();
        });

        it('should render the button with full width styling class', () => {
            const { container } = render(<NewSessionButton {...defaultProps} />);
            const button = container.querySelector('.new-session-button');
            expect(button).toBeInTheDocument();
        });
    });

    describe('Always enabled - Requirements: 2.3', () => {
        it('should always be enabled (no disabled prop)', () => {
            render(<NewSessionButton {...defaultProps} />);
            const button = screen.getByRole('button');
            expect(button).not.toBeDisabled();
        });

        it('should show tooltip for creating a new session', () => {
            render(<NewSessionButton {...defaultProps} />);
            const button = screen.getByRole('button');
            expect(button).toHaveAttribute('title', 'Create a new analysis session');
        });
    });

    describe('Click handler', () => {
        it('should call onClick when button is clicked', () => {
            const onClick = vi.fn();
            render(<NewSessionButton onClick={onClick} />);
            fireEvent.click(screen.getByRole('button'));
            expect(onClick).toHaveBeenCalledTimes(1);
        });
    });

    describe('Display', () => {
        it('should always display "New Session" text', () => {
            render(<NewSessionButton {...defaultProps} />);
            expect(screen.getByText('New Session')).toBeInTheDocument();
        });
    });

    describe('Accessibility', () => {
        it('should have an aria-label', () => {
            render(<NewSessionButton {...defaultProps} />);
            const button = screen.getByRole('button');
            expect(button).toHaveAttribute('aria-label', 'Create a new analysis session');
        });

        it('should have aria-hidden on the icon', () => {
            const { container } = render(<NewSessionButton {...defaultProps} />);
            const icon = container.querySelector('.new-session-icon');
            expect(icon).toHaveAttribute('aria-hidden', 'true');
        });
    });
});
