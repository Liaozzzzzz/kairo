import React, { useRef, useState, useEffect, useCallback } from 'react';
import { cn } from '@/lib/utils';

interface ScrollbarProps extends React.HTMLAttributes<HTMLDivElement> {
  height?: string | number;
  maxHeight?: string | number;
  wrapClass?: string;
  viewClass?: string;
  viewStyle?: React.CSSProperties;
  minSize?: number; // Minimum thumb size in px
  header?: React.ReactNode;
  footer?: React.ReactNode;
}

export const Scrollbar = React.forwardRef<HTMLDivElement, ScrollbarProps>(
  (
    {
      height,
      maxHeight,
      className,
      wrapClass,
      viewClass,
      viewStyle,
      children,
      minSize = 20,
      header,
      footer,
      ...props
    },
    ref
  ) => {
    const wrapRef = useRef<HTMLDivElement>(null);
    const thumbRef = useRef<HTMLDivElement>(null);
    const [thumbHeight, setThumbHeight] = useState('0px');
    const [isDragging, setIsDragging] = useState(false);
    const [isHovering, setIsHovering] = useState(false);

    const dragStartY = useRef(0);
    const dragStartTop = useRef(0);

    const updateThumb = useCallback(() => {
      const wrap = wrapRef.current;
      if (!wrap) return;

      const { clientHeight, scrollHeight, scrollTop } = wrap;

      if (scrollHeight <= clientHeight) {
        setThumbHeight('0px');
        return;
      }

      const thumbSize = Math.max((clientHeight / scrollHeight) * clientHeight, minSize);

      // Calculate thumb position ratio
      // The thumb can move within (clientHeight - thumbSize)
      // The scroll content moves within (scrollHeight - clientHeight)
      const maxScrollTop = scrollHeight - clientHeight;
      const maxThumbTop = clientHeight - thumbSize;

      const ratio = scrollTop / maxScrollTop;
      const top = ratio * maxThumbTop;

      setThumbHeight(`${thumbSize}px`);
      // Use translateY for better performance than top
      if (thumbRef.current) {
        thumbRef.current.style.transform = `translateY(${top}px)`;
      }
    }, [minSize]);

    useEffect(() => {
      const wrap = wrapRef.current;
      if (!wrap) return;

      wrap.addEventListener('scroll', updateThumb);
      const observer = new ResizeObserver(updateThumb);
      observer.observe(wrap);
      // Observe the first child (the view) to detect content changes
      if (wrap.firstElementChild) {
        observer.observe(wrap.firstElementChild);
      }

      updateThumb();

      return () => {
        wrap.removeEventListener('scroll', updateThumb);
        observer.disconnect();
      };
    }, [updateThumb, children]);

    const handleMouseDown = (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();

      const thumb = thumbRef.current;
      if (!thumb) return;

      setIsDragging(true);
      dragStartY.current = e.clientY;

      // Get current transform value
      const transform = window.getComputedStyle(thumb).transform;
      const matrix = new DOMMatrix(transform);
      dragStartTop.current = matrix.m42; // translateY value

      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      document.body.style.setProperty('user-select', 'none');
    };

    const handleMouseMove = (e: MouseEvent) => {
      const wrap = wrapRef.current;
      if (!wrap) return;

      const deltaY = e.clientY - dragStartY.current;
      const { clientHeight, scrollHeight } = wrap;
      const thumbSize = parseFloat(thumbHeight);

      const maxThumbTop = clientHeight - thumbSize;
      const maxScrollTop = scrollHeight - clientHeight;

      // Calculate new thumb position
      let newThumbTop = dragStartTop.current + deltaY;
      newThumbTop = Math.max(0, Math.min(maxThumbTop, newThumbTop));

      // Calculate corresponding scroll position
      const scrollRatio = newThumbTop / maxThumbTop;
      wrap.scrollTop = scrollRatio * maxScrollTop;
    };

    const handleMouseUp = () => {
      setIsDragging(false);
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.setProperty('user-select', '');
    };

    // Styles for the wrapper
    const style: React.CSSProperties = {};
    if (height) style.height = height;
    if (maxHeight) style.maxHeight = maxHeight;

    return (
      <div
        ref={ref}
        className={cn('flex flex-col overflow-hidden', className)}
        style={style}
        onMouseEnter={() => setIsHovering(true)}
        onMouseLeave={() => setIsHovering(false)}
        {...props}
      >
        {/* Header Section */}
        {header && <div className="flex-none z-10">{header}</div>}

        {/* Scrollable Area */}
        <div className="flex-1 min-h-0 relative">
          <div
            ref={wrapRef}
            className={cn(
              'h-full w-full overflow-y-auto scrollbar-none', // Hide native scrollbar
              wrapClass
            )}
          >
            <div className={cn(viewClass)} style={viewStyle}>
              {children}
            </div>
          </div>

          {/* Vertical Scrollbar Track/Thumb */}
          <div
            className={cn(
              'absolute right-[2px] top-0 bottom-0 w-1.5 transition-opacity duration-300',
              (isHovering || isDragging) && thumbHeight !== '0px' ? 'opacity-100' : 'opacity-0'
            )}
          >
            <div
              ref={thumbRef}
              className="w-full bg-[#909399]/30 hover:bg-[#909399]/50 rounded-full cursor-pointer transition-colors"
              style={{ height: thumbHeight }}
              onMouseDown={handleMouseDown}
            />
          </div>
        </div>

        {/* Footer Section */}
        {footer && <div className="flex-none z-10">{footer}</div>}
      </div>
    );
  }
);

Scrollbar.displayName = 'Scrollbar';
