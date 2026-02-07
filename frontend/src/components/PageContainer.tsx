import { ReactNode } from 'react';
import { cn } from '@/lib/utils';
import { Scrollbar } from '@/components/Scrollbar';

interface PageContainerProps {
  children: ReactNode;
  className?: string;
  maxWidth?: string;
  header?: ReactNode;
  footer?: ReactNode;
  viewClass?: string;
}

const PageContainer = ({
  children,
  className,
  maxWidth = 'max-w-5xl',
  header,
  footer,
  viewClass,
}: PageContainerProps) => {
  const headerNode = header ? (
    <div className="flex justify-center w-full ">
      <div className={cn('w-full px-10', maxWidth)}>{header}</div>
    </div>
  ) : null;

  const footerNode = footer ? (
    <div className="flex justify-center w-full">
      <div className={cn('w-full px-10', maxWidth)}>{footer}</div>
    </div>
  ) : null;

  return (
    <div className="h-full w-full bg-background text-foreground overflow-hidden">
      <Scrollbar className="h-full w-full py-10" header={headerNode} footer={footerNode}>
        <div className="flex justify-center w-full min-h-full">
          <div className={cn('w-full py-6', maxWidth, className, viewClass)}>{children}</div>
        </div>
      </Scrollbar>
    </div>
  );
};

export default PageContainer;
