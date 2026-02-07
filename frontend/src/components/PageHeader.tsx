import { Typography } from 'antd';

const { Title, Text } = Typography;

const PageHeader = ({ title, subtitle }: { title: string; subtitle?: string }) => (
  <div className="flex items-center justify-between">
    <div className="flex items-center gap-8">
      <div>
        <Title className="!mb-1" level={2}>
          {title}
        </Title>
        {!!subtitle && <Text type="secondary">{subtitle}</Text>}
      </div>
    </div>
  </div>
);

export default PageHeader;
