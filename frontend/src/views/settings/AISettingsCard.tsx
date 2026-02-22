import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, Input, Segmented, Select, Switch, Typography } from 'antd';
import { useSettingStore } from '@/store/useSettingStore';
import { useShallow } from 'zustand/react/shallow';

const { Text } = Typography;

const AISettingsCard = () => {
  const { t } = useTranslation();
  const [aiSegment, setAiSegment] = useState<'analysis' | 'whisper'>('analysis');
  const { ai, setAI, whisperAi, setWhisperAI } = useSettingStore(
    useShallow((state) => ({
      ai: state.ai,
      setAI: state.setAI,
      whisperAi: state.whisperAi,
      setWhisperAI: state.setWhisperAI,
    }))
  );

  return (
    <Card
      variant="borderless"
      size="small"
      title={
        <span className="text-base font-semibold text-gray-800 dark:text-gray-200">
          {t('settings.tabs.ai')}
        </span>
      }
    >
      <div className="px-2 py-0 flex flex-col gap-4">
        <Segmented
          value={aiSegment}
          block
          onChange={(val) => setAiSegment(val as 'analysis' | 'whisper')}
          options={[
            { label: t('settings.ai.analysisTitle'), value: 'analysis' },
            { label: t('settings.ai.whisperTitle'), value: 'whisper' },
          ]}
        />
        {aiSegment === 'analysis' && (
          <>
            <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
              <div className="md:col-span-4">
                <Text strong className="block text-[13px] text-gray-600 dark:text-gray-400 mb-0">
                  {t('settings.ai.enabled')}
                </Text>
              </div>
              <div className="md:col-span-8">
                <Switch
                  checked={ai.enabled}
                  onChange={(checked) => setAI({ ...ai, enabled: checked })}
                />
              </div>
            </div>

            {ai.enabled && (
              <>
                <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
                  <div className="md:col-span-4">
                    <Text
                      strong
                      className="block text-[13px] text-gray-600 dark:text-gray-400 mb-0"
                    >
                      {t('settings.ai.provider')}
                    </Text>
                  </div>
                  <div className="md:col-span-8">
                    <Select
                      value={ai.provider}
                      onChange={(val) => {
                        let baseUrl = ai.baseUrl;
                        let modelName = ai.modelName;
                        if (val === 'openai') {
                          baseUrl = 'https://api.openai.com/v1';
                          modelName = 'gpt-3.5-turbo';
                        } else if (val === 'anthropic') {
                          baseUrl = 'https://api.anthropic.com/v1';
                          modelName = 'claude-3-opus-20240229';
                        } else if (val === 'local') {
                          baseUrl = 'http://localhost:11434/v1';
                          modelName = 'llama3';
                        } else if (val === 'deepseek') {
                          baseUrl = 'https://api.deepseek.com';
                          modelName = 'deepseek-chat';
                        } else if (val === 'siliconflow') {
                          baseUrl = 'https://api.siliconflow.cn/v1';
                          modelName = 'deepseek-ai/DeepSeek-V3';
                        }
                        setAI({ ...ai, provider: val, baseUrl, modelName });
                      }}
                      style={{ width: '100%' }}
                      options={[
                        { value: 'openai', label: 'OpenAI' },
                        { value: 'anthropic', label: 'Anthropic' },
                        { value: 'gemini', label: 'Google Gemini' },
                        { value: 'deepseek', label: 'DeepSeek' },
                        { value: 'siliconflow', label: 'SiliconFlow (硅基流动)' },
                        { value: 'local', label: 'Local (Ollama/Compatible)' },
                        { value: 'custom', label: 'Custom' },
                      ]}
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
                  <div className="md:col-span-4">
                    <Text
                      strong
                      className="block text-[13px] text-gray-600 dark:text-gray-400 mb-0"
                    >
                      {t('settings.ai.baseUrl')}
                    </Text>
                  </div>
                  <div className="md:col-span-8">
                    <Input
                      value={ai.baseUrl}
                      onChange={(e) => setAI({ ...ai, baseUrl: e.target.value })}
                      placeholder="https://api.openai.com/v1"
                      className="dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
                  <div className="md:col-span-4">
                    <Text
                      strong
                      className="block text-[13px] text-gray-600 dark:text-gray-400 mb-0"
                    >
                      {t('settings.ai.apiKey')}
                    </Text>
                  </div>
                  <div className="md:col-span-8">
                    <Input.Password
                      value={ai.apiKey}
                      onChange={(e) => setAI({ ...ai, apiKey: e.target.value })}
                      placeholder="sk-..."
                      className="dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
                  <div className="md:col-span-4">
                    <Text
                      strong
                      className="block text-[13px] text-gray-600 dark:text-gray-400 mb-0"
                    >
                      {t('settings.ai.modelName')}
                    </Text>
                  </div>
                  <div className="md:col-span-8">
                    <Input
                      value={ai.modelName}
                      onChange={(e) => setAI({ ...ai, modelName: e.target.value })}
                      placeholder="gpt-3.5-turbo"
                      className="dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-start">
                  <div className="md:col-span-4 pt-1">
                    <Text
                      strong
                      className="block text-[13px] text-gray-600 dark:text-gray-400 mb-0"
                    >
                      {t('settings.ai.prompt')}
                    </Text>
                  </div>
                  <div className="md:col-span-8">
                    <Input.TextArea
                      value={ai.prompt}
                      onChange={(e) => setAI({ ...ai, prompt: e.target.value })}
                      placeholder={t('settings.ai.promptPlaceholder')}
                      rows={3}
                      className="dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700"
                    />
                  </div>
                </div>
              </>
            )}
          </>
        )}

        {aiSegment === 'whisper' && (
          <>
            <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
              <div className="md:col-span-4">
                <Text strong className="block text-[13px] text-gray-600 dark:text-gray-400 mb-0">
                  {t('settings.ai.whisperEnabled')}
                </Text>
              </div>
              <div className="md:col-span-8">
                <Switch
                  checked={whisperAi.enabled}
                  onChange={(checked) => setWhisperAI({ ...whisperAi, enabled: checked })}
                />
              </div>
            </div>

            {whisperAi.enabled && (
              <>
                <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
                  <div className="md:col-span-4">
                    <Text
                      strong
                      className="block text-[13px] text-gray-600 dark:text-gray-400 mb-0"
                    >
                      {t('settings.ai.provider')}
                    </Text>
                  </div>
                  <div className="md:col-span-8">
                    <Select
                      value={whisperAi.provider}
                      onChange={(val) => {
                        let baseUrl = whisperAi.baseUrl;
                        let modelName = whisperAi.modelName;
                        if (val === 'openai') {
                          baseUrl = 'https://api.openai.com/v1';
                          modelName = 'whisper-1';
                        } else if (val === 'anthropic') {
                          baseUrl = 'https://api.anthropic.com/v1';
                          modelName = 'whisper-1';
                        } else if (val === 'local') {
                          baseUrl = 'http://localhost:11434/v1';
                          modelName = 'whisper-1';
                        } else if (val === 'deepseek') {
                          baseUrl = 'https://api.deepseek.com';
                          modelName = 'whisper-1';
                        } else if (val === 'siliconflow') {
                          baseUrl = 'https://api.siliconflow.cn/v1';
                          modelName = 'whisper-1';
                        }
                        setWhisperAI({ ...whisperAi, provider: val, baseUrl, modelName });
                      }}
                      style={{ width: '100%' }}
                      options={[
                        { value: 'openai', label: 'OpenAI' },
                        { value: 'anthropic', label: 'Anthropic' },
                        { value: 'gemini', label: 'Google Gemini' },
                        { value: 'deepseek', label: 'DeepSeek' },
                        { value: 'siliconflow', label: 'SiliconFlow (硅基流动)' },
                        { value: 'local', label: 'Local (Ollama/Compatible)' },
                        { value: 'custom', label: 'Custom' },
                      ]}
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
                  <div className="md:col-span-4">
                    <Text
                      strong
                      className="block text-[13px] text-gray-600 dark:text-gray-400 mb-0"
                    >
                      {t('settings.ai.baseUrl')}
                    </Text>
                  </div>
                  <div className="md:col-span-8">
                    <Input
                      value={whisperAi.baseUrl}
                      onChange={(e) => setWhisperAI({ ...whisperAi, baseUrl: e.target.value })}
                      placeholder="https://api.openai.com/v1"
                      className="dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
                  <div className="md:col-span-4">
                    <Text
                      strong
                      className="block text-[13px] text-gray-600 dark:text-gray-400 mb-0"
                    >
                      {t('settings.ai.apiKey')}
                    </Text>
                  </div>
                  <div className="md:col-span-8">
                    <Input.Password
                      value={whisperAi.apiKey}
                      onChange={(e) => setWhisperAI({ ...whisperAi, apiKey: e.target.value })}
                      placeholder="sk-..."
                      className="dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
                  <div className="md:col-span-4">
                    <Text
                      strong
                      className="block text-[13px] text-gray-600 dark:text-gray-400 mb-0"
                    >
                      {t('settings.ai.modelName')}
                    </Text>
                  </div>
                  <div className="md:col-span-8">
                    <Input
                      value={whisperAi.modelName}
                      onChange={(e) => setWhisperAI({ ...whisperAi, modelName: e.target.value })}
                      placeholder="whisper-1"
                      className="dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700"
                    />
                  </div>
                </div>

                {/* <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
                  <div className="md:col-span-4">
                    <Text
                      strong
                      className="block text-[13px] text-gray-600 dark:text-gray-400 mb-0"
                    >
                      {t('settings.ai.language')}
                    </Text>
                  </div>
                  <div className="md:col-span-8">
                    <Input
                      value={whisperAi.language}
                      onChange={(e) => setWhisperAI({ ...whisperAi, language: e.target.value })}
                      placeholder="zh"
                      className="dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700"
                    />
                  </div>
                </div> */}
                <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-start">
                  <div className="md:col-span-4 pt-1">
                    <Text
                      strong
                      className="block text-[13px] text-gray-600 dark:text-gray-400 mb-0"
                    >
                      {t('settings.ai.prompt')}
                    </Text>
                  </div>
                  <div className="md:col-span-8">
                    <Input.TextArea
                      value={whisperAi.prompt}
                      onChange={(e) => setWhisperAI({ ...whisperAi, prompt: e.target.value })}
                      placeholder={t('settings.ai.promptPlaceholder')}
                      rows={3}
                      className="dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700"
                    />
                  </div>
                </div>
              </>
            )}
          </>
        )}
      </div>
    </Card>
  );
};

export default AISettingsCard;
