import React from 'react';
import type { TransitionConfig } from '../types';

interface TransitionOverlayProps {
  isActive: boolean;
  config: TransitionConfig;
}

export const TransitionOverlay: React.FC<TransitionOverlayProps> = ({
  isActive,
  config
}) => {
  if (!isActive) return null;

  // Base transition classes
  const transitionClasses = {
    base: 'absolute inset-0 pointer-events-none bg-black transition-all',
    fade: 'transition-opacity',
    slide: 'transition-transform'
  };

  // Direction-specific transforms for slide transitions
  const slideTransforms = {
    left: 'translate-x-full',
    right: '-translate-x-full',
    up: 'translate-y-full',
    down: '-translate-y-full'
  };

  // Build dynamic classes based on transition type
  const classes = [transitionClasses.base];
  
  if (config.type === 'fade') {
    classes.push(transitionClasses.fade);
  } else if (config.type === 'slide' && config.direction) {
    classes.push(
      transitionClasses.slide,
      slideTransforms[config.direction]
    );
  }

  return (
    <div
      className={classes.join(' ')}
      style={{
        transitionDuration: `${config.duration}ms`
      }}
    />
  );
};
